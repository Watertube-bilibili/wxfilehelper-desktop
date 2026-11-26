package org.ultrafile.lan;

import android.net.Uri;
import org.json.JSONObject;
import java.io.*;
import java.net.*;
import java.security.*;
import java.security.cert.*;
import java.util.*;
import java.util.concurrent.*;
import javax.net.ssl.*;

final class Peer {
    static final int CHUNK=4*1024*1024;
    static final long MAX=8L*1024*1024*1024;
    final String base,token;
    final SSLSocketFactory sockets;
    Peer(String pair)throws Exception{
        URI u=new URI(pair.trim());String frag=u.getFragment();
        if(!"https".equals(u.getScheme())||u.getUserInfo()!=null||u.getPort()<1||u.getPort()>65535||
            !u.getPath().isEmpty()||u.getQuery()!=null||frag==null||!frag.matches("[0-9a-f]{64}\\.[0-9a-f]{32}")||
            u.getHost()==null||!u.getHost().matches("[0-9]+(\\.[0-9]+){3}")||!InetAddress.getByName(u.getHost()).isSiteLocalAddress())
            throw new IOException("请扫描超级文件电脑端的局域网配对码");
        String pin=frag.substring(0,64);token=frag.substring(65);
        base="https://"+u.getHost()+":"+u.getPort()+"/ultra/v1/";
        SSLContext ssl=SSLContext.getInstance("TLS");
        ssl.init(null,new TrustManager[]{new X509TrustManager(){
            public X509Certificate[] getAcceptedIssuers(){return new X509Certificate[0];}
            public void checkClientTrusted(X509Certificate[] chain,String a)throws CertificateException{throw new CertificateException();}
            public void checkServerTrusted(X509Certificate[] chain,String a)throws CertificateException{
                try{if(chain.length>0&&hex(MessageDigest.getInstance("SHA-256").digest(chain[0].getEncoded())).equals(pin))return;}
                catch(GeneralSecurityException e){throw new CertificateException(e);}
                throw new CertificateException("电脑证书与配对码不一致");
            }
        }},null);sockets=ssl.getSocketFactory();
    }
    static String hex(byte[] bytes){StringBuilder s=new StringBuilder();for(byte b:bytes)s.append(String.format(Locale.ROOT,"%02x",b&255));return s.toString();}
    static String hash(File file)throws Exception{
        MessageDigest md=MessageDigest.getInstance("SHA-256");try(InputStream in=new FileInputStream(file)){byte[] b=new byte[65536];int n;while((n=in.read(b))!=-1)md.update(b,0,n);}return hex(md.digest());
    }
    byte[] call(String method,String endpoint,byte[] body,int max)throws IOException{
        HttpsURLConnection c=(HttpsURLConnection)new URL(base+endpoint).openConnection(Proxy.NO_PROXY);
        c.setSSLSocketFactory(sockets);c.setInstanceFollowRedirects(false);c.setConnectTimeout(8000);c.setReadTimeout(60000);
        c.setRequestMethod(method);c.setRequestProperty("Authorization","Bearer "+token);
        try{
            if(body!=null){c.setDoOutput(true);c.setFixedLengthStreamingMode(body.length);try(OutputStream os=c.getOutputStream()){os.write(body);}}
            int status=c.getResponseCode();if(status<200||status>=300)throw new IOException(status==404?"电脑尚未选择文件":"传输失败 HTTP "+status);
            try(InputStream in=c.getInputStream();ByteArrayOutputStream out=new ByteArrayOutputStream()){
                byte[] b=new byte[65536];int n;while((n=in.read(b))!=-1){if(out.size()+n>max)throw new IOException("响应长度异常");out.write(b,0,n);}return out.toByteArray();}
        }finally{c.disconnect();}
    }
    interface Progress{void update(long done,long size);}
    interface Block{void run(int index,int bytes)throws Exception;}
    static void parallel(long size,Block task,Progress report)throws Exception{
        int count=(int)((size+CHUNK-1)/CHUNK);
        ExecutorService pool=Executors.newFixedThreadPool(2);
        java.util.concurrent.atomic.AtomicLong done=new java.util.concurrent.atomic.AtomicLong();
        CompletionService<Void> results=new ExecutorCompletionService<>(pool);
        try{
            // Even and odd blocks travel on separate streams. Retry only the affected block.
            for(int lane=0;lane<2;lane++){final int first=lane;results.submit(()->{
                for(int i=first;i<count;i+=2){int n=(int)Math.min(CHUNK,size-(long)i*CHUNK);
                    for(int attempt=1;;attempt++){if(Thread.interrupted())throw new InterruptedException();try{task.run(i,n);break;}catch(IOException e){if(attempt>=3)throw e;Thread.sleep(250L*attempt);}}
                    report.update(done.addAndGet(n),size);
                }return null;});}
            results.take().get();results.take().get();
        }finally{pool.shutdownNow();pool.awaitTermination(70,TimeUnit.SECONDS);}
    }
    JSONObject metadata()throws Exception{return new JSONObject(new String(call("GET","file",null,4096),"UTF-8"));}
    void download(JSONObject m,File temp,Progress progress)throws Exception{
        long size=m.getLong("Size");if(size<0||size>MAX)throw new IOException("文件大小超限");
        String id=Uri.encode(m.getString("ID"));parallel(size,(i,n)->{
            byte[] b=call("GET","chunk?id="+id+"&i="+i,null,n);if(b.length!=n)throw new IOException("数据块不完整");
            try(RandomAccessFile f=new RandomAccessFile(temp,"rw")){f.seek((long)i*CHUNK);f.write(b);}
        },progress);
        if(!hash(temp).equals(m.getString("SHA256")))throw new IOException("文件校验失败");
    }
    void upload(File temp,String name,Progress progress)throws Exception{
        long size=temp.length();JSONObject m=new JSONObject().put("Name",name).put("Size",size).put("SHA256",hash(temp));
        JSONObject accepted=new JSONObject(new String(call("POST","start",m.toString().getBytes("UTF-8"),4096),"UTF-8"));
        String id=Uri.encode(accepted.getString("ID"));
        try{
            parallel(size,(i,n)->{byte[] b=new byte[n];try(RandomAccessFile f=new RandomAccessFile(temp,"r")){f.seek((long)i*CHUNK);f.readFully(b);}call("PUT","chunk?id="+id+"&i="+i,b,128);},progress);
            call("POST","finish?id="+id,new byte[0],128);
        }catch(Exception e){try{call("DELETE","upload?id="+id,null,128);}catch(IOException ignored){}throw e;}
    }
}

