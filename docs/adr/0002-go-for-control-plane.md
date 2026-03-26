# ADR 0002: Go para control plane

- `Status`: aceito
- `Contexto`: o control plane precisa integrar servicos, consumir eventos, persistir estado e expor API local com boa ergonomia operacional.
- `Decisao`: usar `Go` no backend principal.
- `Consequencias`: operacao simples, binario enxuto e boa concorrencia para workloads de orchestracao.

