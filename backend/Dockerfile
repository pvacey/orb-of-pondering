# Use an official Python runtime as a parent image
FROM python:3.12-slim

# Set the working directory in the container
WORKDIR /app

# Install uv
RUN pip install uv

# Copy the dependency files
COPY pyproject.toml uv.lock ./

# Install dependencies using uv
RUN uv sync --locked

# Copy the local source code to the container
COPY main.py .

# Command to run the application
# Expose port 8000 for the API
EXPOSE 8000
CMD ["uv", "run", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
