#!/usr/bin/env bash
# Convert SEA-LION 9B HF -> MLX (one-time), train the LoRA, and fuse the adapter.
set -euo pipefail
cd "$(dirname "$0")/.."   # repo root

HF="aisingapore/Gemma-SEA-LION-v3-9B-IT"
MLX="training/models/sea-lion-9b-mlx"
ADAPT="training/adapters"
FUSED="training/fused/sea-lion-taglish"
MLXLM="training/.venv/bin"

if [ ! -d "$MLX" ]; then
  echo ">> converting $HF -> $MLX (bf16, ~18GB download, one-time)"
  "$MLXLM/mlx_lm.convert" --model "$HF" --mlx-path "$MLX"
fi

echo ">> training LoRA"
"$MLXLM/mlx_lm.lora" --config training/lora_config.yaml

echo ">> fusing adapter -> $FUSED"
"$MLXLM/mlx_lm.fuse" --model "$MLX" --adapter-path "$ADAPT" --save-path "$FUSED"

echo ">> done: fused model at $FUSED"
