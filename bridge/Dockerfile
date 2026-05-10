FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY audio_bridge.py .
EXPOSE 11235
CMD ["python", "audio_bridge.py"]
