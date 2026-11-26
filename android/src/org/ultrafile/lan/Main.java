package org.ultrafile.lan;

import android.Manifest;
import android.app.Activity;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.database.Cursor;
import android.graphics.Typeface;
import android.net.Uri;
import android.os.Bundle;
import android.provider.OpenableColumns;
import android.view.View;
import android.widget.*;
import java.io.*;
import java.util.concurrent.*;
import org.json.JSONObject;

public class Main extends Activity {
    private static final int CAMERA = 10, SCAN = 11, PICK = 12, SAVE = 13;
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private EditText link;
    private TextView status;
    private Button send, receive, scan, connect;
    private ProgressBar progress;
    private Peer peer;
    private JSONObject offer;
    private boolean busy;
    private long lastProgress;
    private int dp(int n) { return Math.round(n * getResources().getDisplayMetrics().density); }
    private Button button(LinearLayout box, String title, View.OnClickListener listener) {
        Button b = new Button(this); b.setAllCaps(false); b.setText(title); b.setOnClickListener(listener); box.addView(b); return b;
    }
    public void onCreate(Bundle state) {
        super.onCreate(state);
        ScrollView scroller = new ScrollView(this);
        LinearLayout box = new LinearLayout(this); box.setOrientation(LinearLayout.VERTICAL);
        box.setPadding(dp(20),dp(32),dp(20),dp(24)); scroller.addView(box); setContentView(scroller);
        if (android.os.Build.VERSION.SDK_INT >= 30) scroller.setOnApplyWindowInsetsListener((v,insets)->{
            android.graphics.Insets bars=insets.getInsets(android.view.WindowInsets.Type.systemBars());
            v.setPadding(bars.left,bars.top,bars.right,bars.bottom);return insets;});
        TextView heading = new TextView(this); heading.setText("超级文件"); heading.setTextSize(32);
        heading.setTypeface(Typeface.DEFAULT,Typeface.BOLD); box.addView(heading);
        TextView help = new TextView(this); help.setText("在同一 Wi-Fi 下，打开电脑端「超级文件直传」，点「启用局域网连接」。");
        help.setPadding(0,dp(12),0,dp(12)); box.addView(help);
        scan = button(box,"扫码配对",v->scan());
        link = new EditText(this); link.setSingleLine(); link.setHint("或粘贴电脑上的配对信息");
        link.setSaveEnabled(false); link.setInputType(0x81); box.addView(link);
        connect = button(box,"使用配对信息",v->pair(link.getText().toString()));
        send = button(box,"发送文件到电脑",v->startActivityForResult(new Intent(Intent.ACTION_OPEN_DOCUMENT)
            .addCategory(Intent.CATEGORY_OPENABLE).setType("*/*"),PICK));
        receive = button(box,"接收电脑上的文件",v->run(()->{
            JSONObject m=peer.metadata();
            runOnUiThread(()->{if(isDestroyed())return; offer=m;
                Intent dest=new Intent(Intent.ACTION_CREATE_DOCUMENT).addCategory(Intent.CATEGORY_OPENABLE).setType("application/octet-stream");
                dest.putExtra(Intent.EXTRA_TITLE,new File(m.optString("Name","file")).getName());startActivityForResult(dest,SAVE);});
            return "请选择保存位置";}));
        progress = new ProgressBar(this,null,android.R.attr.progressBarStyleHorizontal); box.addView(progress);
        status = new TextView(this); status.setPadding(0,dp(12),0,0); box.addView(status);
        status.setText("等待配对"); controls();
    }
    private void controls(){
        link.setEnabled(!busy); scan.setEnabled(!busy); connect.setEnabled(!busy);
        send.setEnabled(!busy && peer!=null); receive.setEnabled(!busy && peer!=null);
        progress.setVisibility(busy?View.VISIBLE:View.INVISIBLE);
    }
    private void pair(String value) {
        try { peer=new Peer(value); link.setText(""); status.setText("已读取配对信息，可以收发文件。"); }
        catch(Exception e){peer=null;status.setText("配对信息无效，请重新扫描电脑端二维码。");}
        controls();
    }
    private void scan(){if(checkSelfPermission(Manifest.permission.CAMERA)==PackageManager.PERMISSION_GRANTED)
        startActivityForResult(new Intent(this,Scanner.class),SCAN);
        else requestPermissions(new String[]{Manifest.permission.CAMERA},CAMERA);}
    public void onRequestPermissionsResult(int id,String[] p,int[] grants){super.onRequestPermissionsResult(id,p,grants);
        if(id==CAMERA && grants.length>0 && grants[0]==PackageManager.PERMISSION_GRANTED)scan();
        else status.setText("可以在输入框粘贴电脑端的配对信息。");}
    private void run(Callable<String> job){
        if(busy)return;busy=true;progress.setProgress(0);progress.setIndeterminate(true);lastProgress=0;
        status.setText("正在处理…");controls();
        worker.execute(()->{String result;
            try{result=job.call();}catch(Exception e){Throwable c=e;while(c.getCause()!=null)c=c.getCause();result="未完成 · "+c.getLocalizedMessage();}
            final String message=result;runOnUiThread(()->{if(!isDestroyed()){busy=false;status.setText(message);controls();}});});
    }
    private synchronized void transferred(long done,long size){
        long now=android.os.SystemClock.elapsedRealtime(); if(now-lastProgress<160 && done!=size)return; lastProgress=now;
        int pct=(int)(done*100/Math.max(1,size));runOnUiThread(()->{progress.setIndeterminate(false);progress.setProgress(pct);status.setText("正在传输 "+pct+"%");});
    }
    private static void copy(InputStream input,OutputStream output)throws IOException{
        try(InputStream in=input;OutputStream out=output){
            if(in==null||out==null)throw new IOException("无法访问所选文件");
            byte[] block=new byte[65536];long size=0;int n;
            while((n=in.read(block))!=-1){if(Thread.currentThread().isInterrupted())throw new InterruptedIOException();
                size+=n;if(size>Peer.MAX)throw new IOException("单个文件上限为 8 GiB");out.write(block,0,n);}
        }
    }
    private String filename(Uri uri){try(Cursor c=getContentResolver().query(uri,new String[]{OpenableColumns.DISPLAY_NAME},null,null,null)){
        if(c!=null&&c.moveToFirst()&&!c.isNull(0))return c.getString(0);}return "file";}
    protected void onActivityResult(int id,int code,Intent data){
        super.onActivityResult(id,code,data);if(code!=RESULT_OK||data==null)return;
        if(id==SCAN){pair(data.getStringExtra("pair"));return;}
        Uri uri=data.getData();if(uri==null||peer==null)return;
        if(id==PICK)run(()->{String name=filename(uri);File temp=File.createTempFile("send-",null,getCacheDir());
            try{copy(getContentResolver().openInputStream(uri),new FileOutputStream(temp));
                peer.upload(temp,name,this::transferred);return "电脑已接收 · "+name;}finally{temp.delete();}});
        if(id==SAVE&&offer!=null){JSONObject m=offer;offer=null;run(()->{
            File temp=File.createTempFile("receive-",null,getCacheDir());try{
                peer.download(m,temp,this::transferred);copy(new FileInputStream(temp),getContentResolver().openOutputStream(uri,"wt"));
                return "已校验并保存 · "+m.getString("Name");}finally{temp.delete();}});}
    }
    protected void onDestroy(){worker.shutdownNow();super.onDestroy();}
}

