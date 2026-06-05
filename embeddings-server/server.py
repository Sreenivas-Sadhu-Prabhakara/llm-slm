"""OpenAI-compatible /v1/embeddings server for BGE-M3 (1024-dim dense)."""
from fastapi import FastAPI
from pydantic import BaseModel
from FlagEmbedding import BGEM3FlagModel

app = FastAPI()
model = BGEM3FlagModel("BAAI/bge-m3", use_fp16=True)


class EmbReq(BaseModel):
    input: list[str] | str
    model: str = "bge-m3"


@app.get("/health")
def health():
    return {"status": "healthy", "model": "bge-m3", "dim": 1024}


@app.post("/v1/embeddings")
def embeddings(req: EmbReq):
    texts = [req.input] if isinstance(req.input, str) else req.input
    vecs = model.encode(texts, batch_size=8)["dense_vecs"]
    data = [
        {"object": "embedding", "index": i, "embedding": v.tolist()}
        for i, v in enumerate(vecs)
    ]
    return {"object": "list", "data": data, "model": "bge-m3",
            "usage": {"prompt_tokens": 0, "total_tokens": 0}}
