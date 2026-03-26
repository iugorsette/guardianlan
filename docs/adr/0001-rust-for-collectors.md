# ADR 0001: Rust para collectors

- `Status`: aceito
- `Contexto`: collectors lidam com ingestao concorrente, baixo uso de recursos e possiveis caminhos futuros de captura com mais privilegios.
- `Decisao`: usar `Rust` como base dos collectors.
- `Consequencias`: maior previsibilidade de memoria e bom caminho para performance, com maior custo de implementacao do que linguagens mais dinamicas.

