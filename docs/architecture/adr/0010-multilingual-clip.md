# ADR-0010 Multilingual CLIP for text to image matching

Status: Accepted, 2026-10-17

## Context

Owners rarely have a photo of what they lost. Finders almost always photograph what they
found. So the most common pair is an Indonesian description on one side and a photo on the
other. Image to image similarity cannot score it, and text to text similarity misses
everything the finder did not write down.

## Decision

Encode images with `sentence-transformers/clip-ViT-B-32` and lost report text with
`sentence-transformers/clip-ViT-B-32-multilingual-v1`, a multilingual text encoder
distilled into the same embedding space. Their cosine is the visual component in mode
`cross_modal` with its own normalisation band and a lower weight (matching-engine
sections 6.1 and 7). A separate multilingual sentence encoder, `multilingual-e5-small`,
handles text to text similarity.

## Consequences

- A lost report without a photo still gets a visual signal, which the evaluation measures
  as its own slice with its own gate.
- Two text encoders run per report. Both are small and together cost tens of milliseconds
  on CPU.
- The image encoder and the multilingual text encoder must always change together,
  because they share a space. The model lock file pins them as a pair.

## Alternatives

- **Translate Indonesian text to English and use the original CLIP text encoder.** Adds a
  translation model, its latency and its errors.
- **A larger multilingual CLIP such as XLM-R based ViT-L/14 or a SigLIP variant.** Better
  accuracy, but several times the CPU cost per image. Kept as a candidate for a later model
  version if the CPU budget allows.
- **Image captioning followed by text matching.** Two lossy steps instead of one.
