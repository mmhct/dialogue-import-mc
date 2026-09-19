// Runs against an isolated Paper 1.21.1 + WorldEdit 7.3.8 test server.
// Dev dependency: npm install --prefix .validation/node mineflayer
// This script never connects to a public Minecraft server.
const fs=require('fs'),path=require('path'),net=require('net'),assert=require('assert'),cp=require('child_process');
const root=path.resolve(__dirname,'..');
const mineflayer=require(path.join(root,'.validation/node/node_modules/mineflayer'));
const {Vec3}=require(path.join(root,'.validation/node/node_modules/vec3'));
const sleep=ms=>new Promise(r=>setTimeout(r,ms));
class Rcon {
  constructor(){this.pending=new Map();this.id=10;this.buffer=Buffer.alloc(0);}
  async connect(){this.socket=net.createConnection({host:'127.0.0.1',port:25589});this.socket.on('data',b=>{this.buffer=Buffer.concat([this.buffer,b]);while(this.buffer.length>=4){const n=this.buffer.readInt32LE(0);if(this.buffer.length<n+4)return;const p=this.buffer.subarray(4,n+4);this.buffer=this.buffer.subarray(n+4);const id=p.readInt32LE(0),type=p.readInt32LE(4),text=p.subarray(8,-2).toString('utf8');const item=this.pending.get(id);if(item&&(!item.auth||type===2)){clearTimeout(item.timer);this.pending.delete(id);item.resolve(text);}}});await new Promise((r,j)=>{this.socket.once('connect',r);this.socket.once('error',j)});await this.send('dialogueforge-local-test',3);}
  send(text,type=2){return new Promise((resolve,reject)=>{const id=++this.id,b=Buffer.from(text,'utf8'),p=Buffer.alloc(14+b.length);p.writeInt32LE(10+b.length,0);p.writeInt32LE(id,4);p.writeInt32LE(type,8);b.copy(p,12);const timer=setTimeout(()=>{this.pending.delete(id);reject(Error('RCON timeout: '+text))},20000);this.pending.set(id,{resolve,reject,timer,auth:type===3});this.socket.write(p);});}
}
async function waitUntil(test,timeout=15000){const start=Date.now();while(Date.now()-start<timeout){if(test())return;await sleep(100);}throw Error('Timed out waiting for Minecraft result');}
async function main(){
 const allCases=[
  {name:'single',width:8,delays:[20]},
  {name:'fast_folds',width:8,delays:Array(18).fill(1)},
  {name:'mixed_folds',width:8,delays:[1,3,4,7,20,37,9,11,5,6]},
  {name:'medium',width:16,delays:Array(12).fill(10)},
  {name:'wide',width:64,delays:Array(6).fill(20)},
  {name:'chunk_crossing',width:8,delays:Array(24).fill(20)},
  {name:'unicode_storage',width:8,delays:[20],unicode:true},
 ];
 const cases=allCases.filter(c=>!process.env.VALIDATION_CASE||c.name===process.env.VALIDATION_CASE);
 assert(cases.length,'Unknown VALIDATION_CASE');
 const tickRate=Number(process.env.VALIDATION_TICK_RATE||100);
 const schemDir=path.join(root,'.validation/server/plugins/WorldEdit/schematics');fs.mkdirSync(schemDir,{recursive:true});
 for(const c of cases){const request={name:c.name+' 😀',max_width:c.width,target:'@a',lines:c.delays.map((d,i)=>({text:`[${c.name}:${i}] ${i%2?'B':'A'}：对白 "引号" \\ 路径`+(c.unicode?' 😀':''),source_line:i+1,delay_tenths:d}))};const project=path.join(root,'.validation',c.name+'.json'),preview=path.join(root,'.validation',c.name+'.layout.json');fs.writeFileSync(project,JSON.stringify(request));cp.execFileSync(path.join(root,'.validation/dialogueforge'),['--project',project,'--output',path.join(schemDir,c.name+'.schem'),'--preview-json',preview]);c.request=request;c.layout=JSON.parse(fs.readFileSync(preview));}
 const rcon=new Rcon();await rcon.connect();const messages=[];
 const bot=mineflayer.createBot({host:'127.0.0.1',port:25579,username:'DialogueTest',auth:'offline',version:'1.21.1',physicsEnabled:false});
 // Keep the paste origin fixed in the air; client gravity otherwise shifts it.
 bot.physicsEnabled=false;
 bot.on('message',m=>messages.push(m.toString()));bot.on('error',e=>console.error('BOT',e));
 await new Promise((resolve,reject)=>{bot.once('spawn',resolve);bot.once('kicked',reject);setTimeout(()=>reject(Error('Bot spawn timeout')),30000)});
 bot.physicsEnabled=false;
 const results=[];
 try{
 await rcon.send('op DialogueTest');await rcon.send('gamemode creative DialogueTest');await rcon.send('gamerule commandBlockOutput false');await rcon.send('gamerule doMobSpawning false');await rcon.send('gamerule doDaylightCycle false');await rcon.send(`tick rate ${tickRate}`);
 for(let ci=0;ci<cases.length;ci++){
  const c=cases[ci],bx=ci*96,bz=0,by=80;
  await rcon.send(`forceload add ${bx} ${bz} ${bx+c.layout.width} ${bz+c.layout.length}`);
  await rcon.send(`fill ${bx} ${by-5} ${bz} ${bx+c.layout.width} ${by+5} ${bz+c.layout.length} air`);
  await rcon.send(`tp DialogueTest ${bx} ${by} ${bz}`);await sleep(250);
  let at=messages.length;bot.chat('//schem load '+c.name);await waitUntil(()=>messages.slice(at).some(m=>/loaded/i.test(m)));console.log(c.name,'loaded');
  at=messages.length;bot.chat('//paste');await waitUntil(()=>messages.slice(at).some(m=>/pasted/i.test(m)));console.log(c.name,'pasted',messages.slice(at),bot.entity.position);
  await rcon.send(`tp DialogueTest ${bx+.5} ${by+2} ${bz+1.5}`);await bot.waitForChunksToLoad();await sleep(250);
  const button=bot.blockAt(new Vec3(bx,by+1,bz));assert(button&&button.name==='stone_button','button missing at '+[bx,by+1,bz]+': '+button?.name);
  if(c.unicode){
   // Prismarine's chat NBT decoder uses normal UTF-8 and corrupts surrogate
   // pairs. Read the actual stored command over RCON, whose transport is UTF-8.
   const stored=await rcon.send(`data get block ${bx+1} ${by} ${bz} Command`);
   assert(stored.includes('😀')&&!stored.includes('\ufffd'),'Unicode command changed: '+stored);
   results.push({case:c.name,pass:true,stored_command_response:stored});console.log('PASS Unicode command NBT',stored);continue;
  }
  for(let play=0;play<2;play++){
   const start=messages.length;await bot.activateBlock(button,new Vec3(0,1,0));
   await waitUntil(()=>messages.slice(start).some(m=>m===c.request.lines.at(-1).text),Math.max(15000,c.layout.duration_tenths*2000/tickRate+5000));await sleep(Math.max(350,24000/tickRate));
   const observed=messages.slice(start).filter(m=>m.startsWith('['+c.name+':'));
   assert.deepStrictEqual(observed,c.request.lines.map(l=>l.text),'out-of-order, skipped, changed or repeated dialogue');
   const actual=[];for(const e of c.layout.events){const response=await rcon.send(`data get block ${bx+e.x} ${by+e.y} ${bz+e.z} LastExecution`);const match=response.match(/(-?\d+)L\s*$/);assert(match,'missing execution: '+response);actual.push(Number(match[1]));}
   const relative=actual.map(t=>t-actual[0]),expected=c.layout.events.map(e=>e.at_tenths*2);assert.deepStrictEqual(relative,expected,'physical timing mismatch');
   results.push({case:c.name,play:play+1,width:c.layout.width,length:c.layout.length,repeaters:c.layout.repeaters,commands:observed.length,relative_game_ticks:relative,pass:true});console.log('PASS',c.name,'play',play+1,JSON.stringify(relative));
  }
 }
 const suffix=process.env.VALIDATION_CASE?'-'+process.env.VALIDATION_CASE+'-'+tickRate:'';
 fs.writeFileSync(path.join(root,'.validation/game-results'+suffix+'.json'),JSON.stringify({server:'Paper 1.21.1 build 133',worldedit:'7.3.8',tick_rate:tickRate,results},null,2));
 } finally {fs.writeFileSync(path.join(root,'.validation/game-chat.json'),JSON.stringify(messages,null,2));await rcon.send('tick rate 20').catch(()=>{});bot.quit();rcon.socket.end();}
}
main().then(()=>process.exit(0)).catch(e=>{console.error(e);process.exit(1)});
