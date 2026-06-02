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
