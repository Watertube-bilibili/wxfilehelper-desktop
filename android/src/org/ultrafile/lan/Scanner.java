package org.ultrafile.lan;

import android.app.*;
import android.content.Intent;
import android.os.Bundle;
import android.hardware.Camera;
import android.view.*;
import android.widget.Toast;
import com.google.zxing.*;
import com.google.zxing.common.HybridBinarizer;

@SuppressWarnings("deprecation")
public class Scanner extends Activity implements SurfaceHolder.Callback, Camera.PreviewCallback {
    private Camera camera;
    private final MultiFormatReader reader = new MultiFormatReader();
    public void onCreate(Bundle state) {
        super.onCreate(state);
        SurfaceView preview = new SurfaceView(this);
        setContentView(preview);
        preview.getHolder().addCallback(this);
    }
    public void surfaceCreated(SurfaceHolder holder) {
        try {
            camera=Camera.open();
            Camera.Parameters p=camera.getParameters();
            if (p.getSupportedFocusModes().contains(Camera.Parameters.FOCUS_MODE_CONTINUOUS_PICTURE)) {
                p.setFocusMode(Camera.Parameters.FOCUS_MODE_CONTINUOUS_PICTURE); camera.setParameters(p);
            }
            camera.setDisplayOrientation(90);
            camera.setPreviewDisplay(holder);
            camera.setPreviewCallback(this);
            camera.startPreview();
        } catch (Exception e) { Toast.makeText(this,"无法使用相机，可粘贴配对信息",Toast.LENGTH_LONG).show(); finish(); }
    }
    public void surfaceDestroyed(SurfaceHolder h) { if(camera!=null){camera.setPreviewCallback(null);camera.release();camera=null;} }
    public void surfaceChanged(SurfaceHolder h,int format,int w,int height) {}
    public void onPreviewFrame(byte[] data, Camera c) {
        Camera.Size size=c.getParameters().getPreviewSize();
        try {
            PlanarYUVLuminanceSource y=new PlanarYUVLuminanceSource(data,size.width,size.height,0,0,size.width,size.height,false);
            String value=reader.decodeWithState(new BinaryBitmap(new HybridBinarizer(y))).getText();
            c.setPreviewCallback(null);setResult(RESULT_OK,new Intent().putExtra("pair",value));finish();
        } catch (ReaderException ignored) {} finally { reader.reset(); }
    }
}

