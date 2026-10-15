package automation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func screenshot(ctx context.Context, tool string, args ...string) (string, error) {
	f, e := os.CreateTemp("", "interpreter-screen-*.png")
	if e != nil {
		return "", e
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)
	if _, e = run(ctx, tool, append(args, name)...); e != nil {
		return "", e
	}
	b, e := os.ReadFile(name)
	if e != nil {
		return "", e
	}
	if len(b) > 6<<20 {
		return "", errors.New("screenshot too large")
	}
	return `{"image":"` + base64.StdEncoding.EncodeToString(b) + `","mime":"image/png"}`, nil
}

// Fixed PowerShell code uses JSON from stdin, never interpolated user code.
func windowsDesktop(ctx context.Context, a Action) (string, error) {
	b, _ := json.Marshal(a)
	script := `$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type @'
using System;using System.Runtime.InteropServices;
public class Input {
 [DllImport("user32.dll")] public static extern bool SetCursorPos(int x,int y);
 [DllImport("user32.dll")] public static extern void mouse_event(uint flags,uint x,uint y,int data,UIntPtr extra);
 [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr w);
 [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr w,uint m,IntPtr a,IntPtr b);
}
'@
$a = [Console]::In.ReadToEnd() | ConvertFrom-Json
switch($a.action){
 'inspect' { $h=[Input]::GetForegroundWindow();Get-Process | Where-Object MainWindowHandle -eq $h | Select-Object ProcessName,MainWindowTitle,Id | ConvertTo-Json -Compress }
 'move' { [Input]::SetCursorPos($a.x,$a.y) }
 'click' { $down=2;$up=4;if($a.button -eq 2){$down=32;$up=64}elseif($a.button -eq 3){$down=8;$up=16}elseif($a.button -gt 3 -or $a.button -lt 0){throw 'invalid button'};[Input]::SetCursorPos($a.x,$a.y)|Out-Null;[Input]::mouse_event($down,0,0,0,[UIntPtr]::Zero);[Input]::mouse_event($up,0,0,0,[UIntPtr]::Zero) }
 'scroll' { [Input]::mouse_event(2048,0,0,($a.amount*120),[UIntPtr]::Zero) }
 'key' { [System.Windows.Forms.SendKeys]::SendWait($a.key) }
 'type' { $s=$a.text -replace '([+^%~(){}\[\]])','{$1}';[System.Windows.Forms.SendKeys]::SendWait($s) }
 'launch' { Start-Process -FilePath $a.application }
 'focus' { $p=Get-Process -Name $a.application | Where-Object MainWindowHandle -ne 0 | Select-Object -First 1;if(!$p){throw 'window unavailable'};[Input]::SetForegroundWindow($p.MainWindowHandle) }
 'close' { [Input]::PostMessage([Input]::GetForegroundWindow(),16,[IntPtr]::Zero,[IntPtr]::Zero) }
 'clipboard_read' { Get-Clipboard -Raw }
 'clipboard_write' { Set-Clipboard -Value $a.text }
 'screenshot' { $r=[System.Windows.Forms.SystemInformation]::VirtualScreen;$b=New-Object Drawing.Bitmap $r.Width,$r.Height;$g=[Drawing.Graphics]::FromImage($b);$g.CopyFromScreen($r.Left,$r.Top,0,0,$b.Size);$ms=New-Object IO.MemoryStream;$b.Save($ms,[Drawing.Imaging.ImageFormat]::Png);@{image=[Convert]::ToBase64String($ms.ToArray());mime='image/png'}|ConvertTo-Json -Compress;$g.Dispose();$b.Dispose();$ms.Dispose() }
 default { throw 'unsupported action' }
}`
	// A temporary script is task-owned and contains no user input or credentials.
	dir, e := os.MkdirTemp("", "interpreter-desktop-*")
	if e != nil {
		return "", e
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "action.ps1")
	if e = os.WriteFile(path, []byte(script), 0600); e != nil {
		return "", e
	}
	return runInput(ctx, string(b), "powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-File", path)
}
func runInput(ctx context.Context, input, name string, args ...string) (string, error) {
	cmd := command(ctx, name, args...)
	cmd.Stdin = strings.NewReader(input)
	var out bounded
	cmd.Stdout = &out
	cmd.Stderr = &bounded{}
	if e := cmd.Run(); e != nil {
		return "", errors.New("native action failed; verify desktop and OS permissions")
	}
	return out.String(), nil
}
