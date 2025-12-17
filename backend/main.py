from strands import Agent
from strands.models.gemini import GeminiModel
from fastapi import FastAPI
from pydantic import BaseModel
import os
from dotenv import load_dotenv

load_dotenv()


model = GeminiModel(
    client_args={
        "api_key": os.getenv("GEMINI_API_KEY")
    },
    # **model_config
    model_id="gemini-2.5-flash",
    params={
        # some sample model parameters
        "temperature": 0.7,
        "max_output_tokens": 2048,
        "top_p": 0.9,
        "top_k": 40
    }
)

app = FastAPI()

agent = Agent(model=model, system_prompt=(
    "You are a mystical orb of pondering that people come to for wisdom. You will provide advice or insight that is deep and relfective but must be extremely concise. Sort of like a prophetic magic 8-ball or a chinese fortune cookie. a wise traveler of the silk road"
),
    callback_handler=None
)


class ReqResp(BaseModel):
    question: str
    wisdom: str | None = None


@app.post("/")
def invoked_cosmic_wisdom(r: ReqResp):
    resp = agent(r.question)
    r.wisdom = resp.message['content'][0]['text']
    return r


def main():
    response = agent("do you think I will win big playing roulette tonight?")


if __name__ == "__main__":
    main()
