# NBA-chatbot

NBA-chatbot implements a Retrieval-Augmented Generation (RAG) chatbot for
NBA statistics.

## Ingest

Ingest ingests NBA statistics and generates embeddings.

Usage:

    ingest file

The [mxbai-embed-large] model generates embeddings for each row in the provided
CSV file. The PostgreSQL database stores these embeddings along with all the
statistical data. The PostgreSQL database requires the [pgvector] extension
to store embeddings.

### Example

Generate and store embeddings for statistics in `stats/player-per-game.csv`:

```sh
$ ingest 'stats/player-per-game.csv'
```

## Server

Server is an HTTP server for NBA statistics.

The `/player-per-game` endpoint returns statistics for the nearest player
to the provided question.

The [mxbai-embed-large] model generates embeddings for questions. The server
queries the PostgreSQL database for related embeddings and statistical
data. The PostgreSQL database requires the [pgvector] extension to query
embeddings. The [llama3](https://ollama.com/library/llama3) model generates
responses to constructed prompts.

[mxbai-embed-large]: https://ollama.com/library/mxbai-embed-large
[pgvector]: https://github.com/pgvector/pgvector
