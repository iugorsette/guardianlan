# ADR 0003: NATS como event bus

- `Status`: aceito
- `Contexto`: a stack e local e precisa de um barramento pequeno, rapido e simples.
- `Decisao`: usar `NATS` como backbone de eventos.
- `Consequencias`: baixo overhead e boa experiencia para eventos internos; retenção longa e replay complexo ficam fora da v1.

