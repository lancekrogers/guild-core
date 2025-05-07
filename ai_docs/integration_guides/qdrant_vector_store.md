# Qdrant Vector Store

Qdrant is used to implement semantic memory via vector embeddings.

## Embedding Schema

- Text chunks are embedded with OpenAI or local models.
- Metadata includes file path, token count, and tags.

## Usage

- Store retrieved context in `PromptChain`.
- Use similarity search for `RAG` before code generation.

## Tuning

- Use cosine distance with a 0.85 similarity threshold.
