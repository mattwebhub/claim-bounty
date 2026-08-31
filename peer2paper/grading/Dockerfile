FROM python:3.13-alpine

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

RUN addgroup -S -g 10001 peer2paper \
    && adduser -S -D -H -u 10001 -G peer2paper peer2paper

WORKDIR /app/peer2paper/grading
COPY --chown=10001:10001 service.py /app/peer2paper/grading/service.py

USER 10001:10001
EXPOSE 8765

ENTRYPOINT ["python3", "/app/peer2paper/grading/service.py", "serve"]
CMD ["--key-root", "/private/keys", "--receipt-root", "/private/receipts", "--token-file", "/private/token/service.token"]
