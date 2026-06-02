# ocr-service

基于 [RapidOCR](https://github.com/RapidAI/RapidOCR) + FastAPI 的轻量级 OCR 微服务，支持 Docker 部署。

## 依赖

```
fastapi==0.111.0
uvicorn==0.30.1
rapidocr-onnxruntime==1.3.22
opencv-python-headless==4.9.0.80
numpy==1.26.4
python-multipart==0.0.9
```

安装：

```bash
pip install -r requirements.txt
```

## 核心代码

```python
from fastapi import FastAPI, File, UploadFile, HTTPException
from rapidocr_onnxruntime import RapidOCR
import numpy as np
import cv2
import uvicorn

app = FastAPI()
engine = RapidOCR()

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/ocr")
async def ocr(file: UploadFile = File(...)):
    contents = await file.read()
    nparr = np.frombuffer(contents, np.uint8)
    img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
    if img is None:
        raise HTTPException(status_code=400, detail="invalid image")
    result, _ = engine(img)
    texts = []
    if result:
        for item in result:
            box = item[0]  # [[x1,y1],[x2,y2],[x3,y3],[x4,y4]]
            y_center = sum(float(p[1]) for p in box) / 4
            x_min = min(float(p[0]) for p in box)
            texts.append({
                "text": item[1],
                "confidence": float(item[2]),
                "y": y_center,
                "x": x_min,
            })
    return {"texts": texts}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8001)
```

## 启动服务

```bash
python main.py
# 服务监听 http://0.0.0.0:8001
```

或使用 Docker：

```bash
docker build -t ocr-service .
docker run -p 8001:8001 ocr-service
```

## 调用示例

**健康检查**

```bash
curl http://localhost:8001/health
# {"status":"ok"}
```

**图片 OCR（curl）**

```bash
curl -X POST http://localhost:8001/ocr \
  -F "file=@/path/to/image.jpg"
```

**返回示例**

```json
{
  "texts": [
    {
      "text": "识别到的文字",
      "confidence": 0.98,
      "y": 120.5,
      "x": 34.2
    }
  ]
}
```

**Python 调用示例**

```python
import requests

with open("image.jpg", "rb") as f:
    resp = requests.post("http://localhost:8001/ocr", files={"file": f})
print(resp.json())
```
