from __future__ import annotations

import os
import sys
import tempfile
from pathlib import Path
from typing import Optional, Tuple, Any, Dict

import numpy as np
import cv2
from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
from insightface.app import FaceAnalysis
from huggingface_hub import snapshot_download

MODEL_DIR = os.getenv("MODEL_DIR", "./models").strip()
PROVIDERS = [p.strip() for p in os.getenv("ORT_PROVIDERS", "CPUExecutionProvider").split(",") if p.strip()]

DET_SIZE = os.getenv("DET_SIZE", "640,640")
DET_SIZE_TUPLE: Tuple[int, int] = tuple(int(x) for x in DET_SIZE.split(",", 1))

GPU_ID = int(os.getenv("GPU_ID", "-1"))

# ---------------------------
# Model root preparation
# ---------------------------

_face_app: Optional[FaceAnalysis] = None

def _load_models() -> FaceAnalysis:
    model_dir = snapshot_download(
        repo_id="fal/AuraFace-v1",
        local_dir=MODEL_DIR
    )

    app = FaceAnalysis(
        name="auraface",
        root=root,
        providers=PROVIDERS,
    )

    # ctx_id: -1 = CPU, 0+ = GPU index in InsightFace terms
    # det_size: influences detector
    app.prepare(ctx_id=GPU_ID, det_size=DET_SIZE_TUPLE)
    return app


def _decode_image(image_bytes: bytes) -> np.ndarray:
    """
    Decode input bytes into a BGR OpenCV image (H, W, 3).
    """
    arr = np.frombuffer(image_bytes, dtype=np.uint8)
    img = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if img is None:
        raise ValueError("Failed to decode image (unsupported format or corrupt file).")
    return img


def _pick_largest_face(faces: list[Any]) -> Any:
    """
    InsightFace returns faces with .bbox = [x1, y1, x2, y2]
    Pick the one with largest area.
    """
    if not faces:
        return None
    best = None
    best_area = -1.0
    for f in faces:
        x1, y1, x2, y2 = [float(v) for v in f.bbox]
        area = max(0.0, x2 - x1) * max(0.0, y2 - y1)
        if area > best_area:
            best_area = area
            best = f
    return best


# ---------------------------
# FastAPI lifecycle
# ---------------------------

@APP.on_event("startup")
def startup() -> None:
    global _face_app
    try:
        _face_app = _load_models()
    except Exception as e:
        # Crash early with a clear message; container orchestrators will restart.
        print(f"[startup] failed to load models: {e}", file=sys.stderr)
        raise

@APP.post("/embed-largest-face")
async def embed_largest_face(file: UploadFile = File(...)) -> JSONResponse:
    """
    Multipart form upload: field name "file"
    Response:
      {
        "embedding": [float...],
        "dim": 512,
        "bbox": [x1,y1,x2,y2],
        "det_score": <float or null>
      }
    """
    global _face_app

    data = await file.read()
    if not data:
        raise HTTPException(status_code=400, detail="Empty file")

    try:
        img = _decode_image(data)
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

    faces = _face_app.get(img)
    face = _pick_largest_face(faces)
    if face is None:
        raise HTTPException(status_code=422, detail="No face detected")

    emb = getattr(face, "embedding", None)
    emb = np.asarray(emb, dtype=np.float32)
    
    # Normalize (cosine similarity expects unit length)
    norm = np.linalg.norm(emb)
    if norm > 0:
        emb = emb / norm

    bbox = [float(v) for v in face.bbox]
    det_score = float(getattr(face, "det_score", 0.0)) if hasattr(face, "det_score") else None

    return JSONResponse(
        {
            "embedding": emb.tolist(),
            "dim": int(emb.shape[0]),
            "bbox": bbox,
            "det_score": det_score,
        }
    )