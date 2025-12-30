from strands import Agent
from strands.models.gemini import GeminiModel
from fastapi import FastAPI, Request
from pydantic import BaseModel, Field
import os
from dotenv import load_dotenv

load_dotenv()


model = GeminiModel(
    client_args={
        "api_key": os.getenv("GEMINI_API_KEY")
    },
    model_id="gemini-3-flash-preview",
    params={
        # some sample model parameters
        "temperature": 0.7,
        "max_output_tokens": 2048,
        "top_p": 0.9,
        "top_k": 40
    }
)

app = FastAPI()


class Inquery(BaseModel):
    question: str = Field(..., examples=[
                          "Will I be too cold without a jacket?"])


class Insight(BaseModel):
    question: str = Field(..., examples=[
                          "Will I be too cold without a jacket?"])
    wisdom: str = Field(..., examples=["The wind honors only the prepared."])


@app.post("/", response_model=Insight)
def seek_cosmic_wisdom(r: Inquery, context: Request):
    agent = Agent(model=model, system_prompt=(
        "You are a mystical orb of pondering that people come to for wisdom. You will provide advice or insight that is deep and relfective but must be extremely concise. Sort of like a prophetic magic 8-ball or a chinese fortune cookie. a wise guru giving spiritual guidance"
    ),
        callback_handler=None
    )
    resp = agent(r.question)
    retval = Insight(
        question=r.question,
        wisdom=resp.message['content'][0]['text']
    )
    print(context.headers)
    print(retval)
    return retval


if __name__ == "__main__":
    main()
