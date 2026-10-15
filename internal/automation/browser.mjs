// Runs one explicitly reviewed workflow in a fresh browser context.
import fs from 'node:fs';
import {pathToFileURL} from 'node:url';
let raw='';for await(const chunk of process.stdin){raw+=chunk;if(raw.length>1048576)throw Error('request too large');}
const request=JSON.parse(raw);
let browser;
try {
 const {chromium}=await import(pathToFileURL(request.module+'/index.mjs').href);
 const allowed=new Set(request.origins.map(x=>new URL(x).origin));
 const check=url=>{const u=new URL(url);return ['http:','https:'].includes(u.protocol)&&!u.username&&!u.password&&allowed.has(u.origin);};
 if(!allowed.size||request.steps.length>30)throw Error('1-30 steps and authorized origins required');
 browser=await chromium.launch({headless:true});
 const context=await browser.newContext({acceptDownloads:false,serviceWorkers:'block'});
 await context.route('**/*',route=>check(route.request().url())?route.continue():route.abort('blockedbyclient'));
 await context.routeWebSocket('**/*',ws=>ws.close());
 const page=await context.newPage();page.setDefaultTimeout(10000);
 const consoleLog=[],errors=[],network=[];
 page.on('console',m=>{if(consoleLog.length<50)consoleLog.push({type:m.type(),text:m.text().slice(0,1000)});});
 page.on('pageerror',e=>{if(errors.length<30)errors.push(e.message.slice(0,1000));});
 page.on('requestfailed',r=>{if(network.length<50)network.push({url:r.url(),error:r.failure()?.errorText});});
 const results=[];
 for(const step of request.steps){
  switch(step.action){
   case 'open':if(!check(step.url))throw Error('URL outside approved origins');await page.goto(step.url,{waitUntil:'domcontentloaded'});break;
   case 'click':await page.locator(step.selector).click();break;
   case 'fill':await page.locator(step.selector).fill(step.text);break;
   case 'press':await page.locator(step.selector).press(step.key);break;
   case 'expect_text':await page.locator(step.selector).filter({hasText:step.text}).first().waitFor({state:'visible'});break;
   case 'expect_url':if(page.url()!==step.url)throw Error('URL assertion failed');break;
   case 'select':await page.locator(step.selector).selectOption(step.value);break;
   case 'screenshot':results.push({action:'screenshot',image:(await page.screenshot({fullPage:false})).toString('base64')});continue;
   default:throw Error('unsupported browser action');
  }
  results.push({action:step.action,status:'passed'});
 }
 const text=(await page.locator('body').innerText()).slice(0,12000);
 console.log(JSON.stringify({ok:true,url:page.url(),title:await page.title(),text,results,console:consoleLog,errors,network}));
} catch(error){console.log(JSON.stringify({ok:false,error:String(error.message).slice(0,1000)}));process.exitCode=1;}
finally{if(browser)await browser.close();}
