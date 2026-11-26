// SPDX-License-Identifier: AGPL-3.0-only
"use strict";
const el = id => document.getElementById(id);
const translatable = [...document.querySelectorAll("[data-en]")];
for (const node of translatable) node.dataset.zh = node.textContent;
let english = false, mode = "receive", selected = null, session = null, downloadURL = "";
const tr = (zh, en) => english ? en : zh;
const visible = (id, on) => { el(id).hidden = !on; };
const sizeText = n => n >= 1048576 ? (n / 1048576).toFixed(1) + " MiB" : n >= 1024 ? (n / 1024).toFixed(1) + " KiB" : n + " B";
const maxFile = 8 * 2 ** 30, chunk = 16384, label = "ultra-file-v1";
const idPattern = /^[a-z2-7]{12}$/;
function randomCode() {
  const alphabet = "abcdefghijklmnopqrstuvwxyz234567";
  return Array.from(crypto.getRandomValues(new Uint8Array(12)), b => alphabet[b & 31]).join("");
}
function setStatus(zh, en) {
  visible("feedback", true);
  el("status").dataset.zh = zh; el("status").dataset.en = en;
  el("status").textContent = tr(zh, en);
}
function updateFile(file) {
  if (session) return;
  selected = file || null;
  el("fileName").textContent = file ? file.name : tr("选择文件","Choose a file");
  el("fileSize").textContent = file ? sizeText(file.size) : tr("也可以把一个文件拖到这里","Or drop one file here");
}
function selectMode(next) {
  mode = next;
  for (const [name, tab] of [["receive","tabReceive"],["send","tabSend"]]) {
    visible(name, name === next); el(tab).setAttribute("aria-selected", String(name === next));
  }
}
function clearResult() {
  if (downloadURL) URL.revokeObjectURL(downloadURL);
  downloadURL = ""; el("save").removeAttribute("href");
  for (const id of ["feedback","save","accept","reject","cancel","progress","copyCode","copyLink"]) visible(id,false);
  el("incoming").textContent = ""; el("meter").textContent = ""; el("code").textContent = "—";
}
function busy(on) {
  for (const id of ["peer","file","offer","listen"]) el(id).disabled = on;
  el("tabReceive").disabled = on && mode !== "receive"; el("tabSend").disabled = on && mode !== "send";
  visible("cancel",on);
}
function udpSDP(value) {
  return {type:value.type, sdp:value.sdp.split("\r\n").filter(line=>!line.startsWith("a=candidate:") || /^a=candidate:\S+ \d+ udp /i.test(line)).join("\r\n")};
}
class Transfer {
  constructor(file, peerCode) {
    this.file=file; this.code=randomCode(); this.peer=file?"ultrafile-"+peerCode:"";
    this.id=file?randomCode():""; this.phase="signaling"; this.total=0;
    this.written=0; this.queue=Promise.resolve(); this.timer=0; this.heartbeat=0;
    this.ready=false; this.queueBytes=0; this.chunks=[]; this.closed=false;
  }
  async start() {
    if (!window.RTCPeerConnection || !crypto.getRandomValues) throw Error(tr("浏览器不支持 WebRTC，请使用新版浏览器。","A WebRTC-capable browser is required."));
    this.ws=new WebSocket("wss://0.peerjs.com/peerjs?"+new URLSearchParams({key:"peerjs",id:"ultrafile-"+this.code,token:randomCode(),version:"1.5.5"}));
    this.timeout(20000,tr("无法连接配对服务，请重试。","Pairing service timed out. Try again."));
    this.ws.onopen=()=>{this.heartbeat=setInterval(()=>{if(this.ws.readyState===1)this.ws.send(JSON.stringify({type:"HEARTBEAT"}));},5000);};
    let signalQueue=Promise.resolve();
    this.ws.onmessage=e=>{signalQueue=signalQueue.then(()=>this.signal(JSON.parse(e.data))).catch(err=>this.fail(err));};
    this.ws.onclose=()=>{if (!this.ready && !this.closed) this.fail(Error(tr("配对连接已断开，请重试。","Pairing connection closed. Try again.")));};
    this.ws.onerror=()=>{if(!this.ready)this.fail(Error(tr("无法连接配对服务。","Cannot reach the pairing service.")));};
  }
  timeout(ms, msg) {clearTimeout(this.timer);this.timer=setTimeout(()=>this.fail(Error(msg)),ms);}
  async signal(m) {
    if(this.closed)return;
    const p=m.payload || {};
    if(m.type==="OPEN") {
      clearTimeout(this.timer);
      if(!this.file) {
        this.phase="waiting"; el("code").textContent=this.code.match(/.{4}/g).join(" ");
        visible("copyCode",true);visible("copyLink",true);
        setStatus("等待发送方连接","Waiting for a sender");return;
      }
      this.makePeer();
      this.bind(this.pc.createDataChannel(label,{ordered:true}));
      await this.describe(await this.pc.createOffer(),"OFFER");
    } else if(m.type==="OFFER" && !this.file && !this.pc && p.label===label && /^ultrafile-[a-z2-7]{12}$/.test(m.src) && p.connectionId) {
      this.peer=m.src; this.id=p.connectionId; this.makePeer();
      this.pc.ondatachannel=e=>{if(e.channel.label===label)this.bind(e.channel);else e.channel.close();};
      await this.pc.setRemoteDescription(udpSDP(p.sdp));
      await this.describe(await this.pc.createAnswer(),"ANSWER");
    } else if(m.type==="ANSWER" && this.pc && m.src===this.peer && this.id===p.connectionId && !this.pc.remoteDescription) {
      await this.pc.setRemoteDescription(udpSDP(p.sdp));
    } else if(m.type==="CANDIDATE" && m.src===this.peer && p.connectionId===this.id && this.pc?.remoteDescription && / udp /i.test(p.candidate?.candidate||"")) {
      await this.pc.addIceCandidate(p.candidate);
    } else if(["EXPIRE","ID-TAKEN","ERROR","INVALID-KEY"].includes(m.type)) {
      throw Error(tr("配对码不可用，请检查接收方是否在线。","Pairing code unavailable. Check that the receiver is online."));
    }
  }
  makePeer() {
    this.pc=new RTCPeerConnection({iceServers:[{urls:"stun:stun.cloudflare.com:3478"}]});
    setStatus("正在建立 UDP 直连…","Connecting over UDP…");
    this.timeout(40000,tr("无法建立 UDP 直连。请切换网络，或让两端连接同一 Wi-Fi。","UDP connection timed out. Try a different network or the same Wi-Fi."));
    this.pc.onconnectionstatechange=()=>{if(this.pc.connectionState==="failed")this.fail(Error(tr("UDP 连接失败。","UDP connection failed.")));};
  }
  async describe(desc,type) {
    await this.pc.setLocalDescription(desc);
    if(this.pc.iceGatheringState!=="complete") await new Promise(resolve=>{
      const pc=this.pc;const finish=()=>{clearTimeout(wait);pc.removeEventListener("icegatheringstatechange",changed);resolve();};
      const changed=()=>{if(pc.iceGatheringState==="complete")finish();};
      const wait=setTimeout(finish,5000);pc.addEventListener("icegatheringstatechange",changed);changed();
    });
    if(this.closed)return;
    this.ws.send(JSON.stringify({type,dst:this.peer,payload:{type:"data",connectionId:this.id,label,serialization:"raw",reliable:true,sdp:udpSDP(this.pc.localDescription)}}));
  }
  bind(dc) {
    this.dc=dc;dc.binaryType="arraybuffer";dc.bufferedAmountLowThreshold=128*1024;
    dc.onopen=()=>{if(this.closed)return;clearTimeout(this.timer);this.ready=true;
      if(this.file){this.phase="offered";this.total=this.file.size;this.control({type:"offer",name:this.file.name,size:this.total});setStatus("等待对方接收","Waiting for acceptance");}
      else {this.phase="connected";setStatus("已连接，等待文件信息","Connected. Waiting for file details");}
    };
    dc.onmessage=e=>{
      const bytes=e.data instanceof ArrayBuffer?e.data.byteLength:0;
      this.queueBytes+=bytes;
      if(this.queueBytes>12*1024*1024){this.fail(Error(tr("接收设备写入过慢，请重试。","Receiver storage is too slow. Try again.")));return;}
      this.queue=this.queue.then(async()=>{if(this.closed)return;if(typeof e.data==="string"){if(e.data.length>4096)throw Error("Invalid metadata");await this.onControl(JSON.parse(e.data));}else await this.receive(e.data);}).finally(()=>{this.queueBytes-=bytes;}).catch(err=>this.fail(err));
    };
    dc.onclose=()=>{if(!this.closed && this.phase!=="complete")this.fail(Error(tr("对方已断开。","Peer disconnected.")));};
    dc.onerror=()=>this.fail(Error(tr("数据通道已中断。","Data channel interrupted.")));
  }
  control(obj){this.dc.send(JSON.stringify(obj));}
  async onControl(m) {
    if(m.type==="offer" && this.phase==="connected") {
      if(!Number.isSafeInteger(m.size)||m.size<0||m.size>maxFile||typeof m.name!=="string"||!m.name||m.name.length>1024)throw Error("Invalid file offer");
      this.phase="incoming";this.total=m.size;this.name=m.name.split(/[/\\]/).pop().replace(/[\x00-\x1f<>:"|?*]/g,"_")||"file";
      el("incoming").textContent=this.name+" · "+sizeText(m.size);setStatus("有文件等待接收","Incoming file");visible("accept",true);visible("reject",true);
    } else if(m.type==="accept" && this.phase==="offered") {
      this.phase="sending";this.send().catch(e=>this.fail(e));
    } else if(m.type==="finish" && this.phase==="receiving") {
      if(this.written!==this.total||this.hash.hex()!==m.hash)throw Error(tr("文件校验失败，请重传。","File checksum mismatch. Retry the transfer."));
      if(this.sink){await this.sink.close();this.sink=null;}else{downloadURL=URL.createObjectURL(new Blob(this.chunks));this.chunks=[];el("save").href=downloadURL;el("save").download=this.name;visible("save",true);}
      this.phase="complete";this.control({type:"ack",hash:m.hash});setStatus("已接收，SHA-256 校验通过","Received and verified with SHA-256");
      this.meter(true);busy(false);visible("copyCode",false);visible("copyLink",false);el("code").textContent="—";
      // Give the reliable channel time to deliver the acknowledgement before cleanup.
      this.dc.onclose=()=>this.close();setTimeout(()=>this.close(),2000);
    } else if(m.type==="ack" && this.phase==="confirming" && m.hash===this.digest) {
      this.phase="complete";setStatus("对方已接收并校验文件","Receiver verified the file");this.meter(true);this.close();
    } else if(m.type==="cancel")this.fail(Error(tr("对方已取消传输。","Peer cancelled the transfer.")));
  }
  async accept() {
    if(this.phase!=="incoming")return;
    if(window.showSaveFilePicker && this.total>32*1048576){try{const dest=await window.showSaveFilePicker({suggestedName:this.name});if(this.closed)return;this.sink=await dest.createWritable();}catch(e){if(e.name==="AbortError")return;throw e;}}
    else if(this.total>128*1048576)throw Error(tr("此浏览器最多接收 128 MiB，请用桌面版或支持直接写入磁盘的浏览器。","This browser can receive up to 128 MiB. Use the desktop app or a browser with direct disk saving."));
    this.hash=sha256.create();this.phase="receiving";visible("accept",false);visible("reject",false);this.control({type:"accept"});setStatus("正在接收…","Receiving…");this.meter(true);
  }
  async receive(buffer) {
    if(this.phase!=="receiving"||!(buffer instanceof ArrayBuffer)||buffer.byteLength>chunk||this.written+buffer.byteLength>this.total)throw Error("Invalid file chunk");
    if(this.sink)await this.sink.write(buffer);else this.chunks.push(buffer);
    this.hash.update(new Uint8Array(buffer));this.written+=buffer.byteLength;this.meter();
  }
  async send() {
    setStatus("正在发送…","Sending…");const hash=sha256.create();
    while(this.written<this.file.size){
      if(this.closed)return;
      if(this.dc.bufferedAmount>512*1024)await new Promise((resolve,reject)=>{const dc=this.dc;const end=()=>{dc.removeEventListener("bufferedamountlow",ready);dc.removeEventListener("close",shut);};const ready=()=>{end();resolve();};const shut=()=>{end();reject(Error("Channel closed"));};dc.addEventListener("bufferedamountlow",ready,{once:true});dc.addEventListener("close",shut,{once:true});if(dc.bufferedAmount<=dc.bufferedAmountLowThreshold)ready();});
      const b=await this.file.slice(this.written,this.written+chunk).arrayBuffer();if(this.closed)return;
      this.dc.send(b);hash.update(new Uint8Array(b));this.written+=b.byteLength;this.meter();
    }
    this.digest=hash.hex();this.phase="confirming";this.control({type:"finish",hash:this.digest});setStatus("等待对方校验文件…","Waiting for receiver verification…");
  }
  meter(force=false){const now=performance.now();if(!force&&now-(this.lastPaint||0)<100)return;this.lastPaint=now;visible("progress",true);el("progress").value=this.total?this.written/this.total:1;el("meter").textContent=sizeText(this.written)+" / "+sizeText(this.total);}
  fail(err){if(this.closed||this.phase==="complete")return;setStatus(err.message||String(err),err.message||String(err));this.close();}
  close(){if(this.closed)return;this.closed=true;clearInterval(this.heartbeat);clearTimeout(this.timer);if(this.sink)this.sink.abort().catch(()=>{});this.chunks=[];this.dc?.close();this.pc?.close();this.ws?.close();if(session===this){session=null;busy(false);["accept","reject","copyCode","copyLink"].forEach(id=>visible(id,false));el("code").textContent="—";}}
}
function begin(file=null) {
  if(session?.phase==="complete")session.close();if(session)return;
  clearResult();let peer="";
  if(file){peer=el("peer").value.toLowerCase().replace(/[\s-]/g,"");if(!idPattern.test(peer)){setStatus("请输入对方的 12 位配对码。","Enter the 12-character receiver code.");return;}
    if(file.size>maxFile){setStatus("每个文件最大 8 GiB。","Maximum file size is 8 GiB.");return;}}
  const t=new Transfer(file,peer);session=t;busy(true);setStatus("连接配对服务…","Connecting to pairing service…");t.start().catch(err=>t.fail(err));
}
for(const [id,m] of [["tabReceive","receive"],["tabSend","send"]])el(id).onclick=()=>{if(!session){clearResult();selectMode(m);}};
el("listen").onclick=()=>begin();el("offer").onclick=()=>selected?begin(selected):setStatus("请先选择文件。","Choose a file first.");
el("file").onchange=e=>updateFile(e.target.files[0]);
const pick=document.querySelector(".pick");pick.ondragover=e=>e.preventDefault();pick.ondrop=e=>{e.preventDefault();updateFile(e.dataTransfer.files[0]);};
el("accept").onclick=()=>session?.accept().catch(e=>session?.fail(e));
function abort(){if(session){try{session.control({type:"cancel"});}catch{}session.close();setStatus("传输已取消","Transfer cancelled");}}
el("reject").onclick=abort;el("cancel").onclick=abort;
for(const id of ["copyCode","copyLink"])el(id).onclick=async()=>{if(!session)return;const txt=id==="copyCode"?session.code:location.href.split("#")[0]+"#to="+session.code;try{await navigator.clipboard.writeText(txt);setStatus("已复制，等待发送方连接","Copied. Waiting for a sender");}catch{setStatus("请手动复制页面上的接收码。","Select and copy the code shown above.");}};
el("lang").onclick=()=>{english=!english;document.documentElement.lang=english?"en":"zh-CN";for(const n of translatable)n.textContent=english?n.dataset.en:n.dataset.zh;el("lang").textContent=english?"简体中文":"English";if(el("status").dataset.zh)el("status").textContent=tr(el("status").dataset.zh,el("status").dataset.en);if(selected){el("fileName").textContent=selected.name;el("fileSize").textContent=sizeText(selected.size);}};
const invited=new URLSearchParams(location.hash.slice(1)).get("to");if(invited&&idPattern.test(invited)){el("peer").value=invited;selectMode("send");}
window.addEventListener("pagehide",()=>session?.close());

