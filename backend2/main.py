import os
import asyncio
import uvicorn
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from dotenv import load_dotenv
import google.generativeai as genai

class SummarizeRequest(BaseModel):
    text: str

class SummarizeResponse(BaseModel):
    summary: str

async def summarize_with_gemini(api_key: str, text: str) -> str:
    if not api_key or not api_key.strip():
        raise ValueError("missing GEMINI_API_KEY")
    genai.configure(api_key=api_key)
    model = genai.GenerativeModel("gemini-2.5-flash")
    prompt = f"Summarize the following text in exactly 3 concise lines. Keep it factual and clear.\n\nTEXT:\n{text}"
    loop = asyncio.get_event_loop()
    def _generate():
        return model.generate_content(prompt)
    resp = await loop.run_in_executor(None, _generate)
    output = "".join([str(p) for c in getattr(resp, "candidates", []) if c and getattr(c, "content", None) for p in c.content.parts]).strip()
    if not output:
        raise RuntimeError("empty response from model")
    raw_lines = [l.strip() for l in output.replace("`", "").split("\n")]
    cleaned = []
    for l in raw_lines:
        s = l.strip("-• ")
        if s:
            cleaned.append(s)
    if not cleaned:
        raise RuntimeError("empty response from model")
    cleaned = cleaned[:3]
    formatted = []
    for s in cleaned:
        t = s.strip()
        if t and not t[0].isupper():
            t = t[0].upper() + t[1:]
        if t and t[-1] not in ".!?":
            t = t + "."
        formatted.append(f"- {t}")
    return "\n".join(formatted)

def create_app() -> FastAPI:
    load_dotenv()
    load_dotenv(dotenv_path=os.path.join(os.path.dirname(__file__), "..", ".env"))
    app = FastAPI()
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["http://localhost:3000"],
        allow_credentials=True,
        allow_methods=["POST", "OPTIONS"],
        allow_headers=["Content-Type", "Authorization"],
    )

    @app.post("/summarize", response_model=SummarizeResponse)
    async def summarize(req: SummarizeRequest, request: Request):
        api_key = os.getenv("GEMINI_API_KEY", "").strip()
        text = (req.text or "").strip()
        if not text:
            raise HTTPException(status_code=400, detail={"error": "text is required"})
        try:
            result = await summarize_with_gemini(api_key, text)
        except ValueError:
            raise HTTPException(status_code=500, detail={"error": "failed to summarize"})
        except Exception:
            raise HTTPException(status_code=500, detail={"error": "failed to summarize"})
        return SummarizeResponse(summary=result)

    return app

app = create_app()

if __name__ == "__main__":
    print("Backend running on http://localhost:4001")
    uvicorn.run(app, host="0.0.0.0", port=4001)