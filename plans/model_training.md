# Custom Model Training & Integration Plan

This document outlines the process for training a custom LLM and integrating it into the Go Backend API.

## 1. Roadmap Overview
The process follows a **Fine-tuning -> Serving -> Integration** pipeline.

### Phase 1: Fine-Tuning (Training)
*   **Base Model:** Recommended Llama-3-8B-Instruct.
*   **Data Format:** JSONL (Instruction/Response pairs).
*   **Tools:** [Unsloth](https://github.com/unslothai/unsloth) (fastest for Mac/Colab) or Axolotl.
*   **Method:** QLoRA (efficient fine-tuning).
*   **Output:** `.gguf` file for inference.

### Phase 2: Serving (Inference)
*   **Local (Mac/Dev):** [Ollama](https://ollama.com/).
    *   Create a `Modelfile`.
    *   `ollama serve` (Exposes API at `http://localhost:11434/v1`).
*   **Production:** [vLLM](https://github.com/vllm-project/vllm) on GPU instances.

### Phase 3: Integration
*   The Go backend uses the `CustomProvider` (implemented in `internal/llm/providers/custom.go`).
*   Configured via environment variables:
    *   `CUSTOM_LLM_BASE_URL`: URL of your inference server.
    *   `CUSTOM_LLM_MODEL`: The name of your fine-tuned model.

## 2. Integration Details
The system is designed with a **Primary -> Fallback** mechanism:
1.  **Primary:** Your custom local model.
2.  **Fallback:** Anthropic Claude (via `ANTHROPIC_API_KEY`).

## 3. Getting Started
1.  Generate a dataset representing your Advocate/Client interactions.
2.  Fine-tune using a Colab notebook with Unsloth.
3.  Download the `.gguf` and load into Ollama.
4.  Update your `.env` file to point to the local server.
