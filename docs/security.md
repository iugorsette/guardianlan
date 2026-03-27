# Security

Esta base foi desenhada para seguranca domestica realista. O foco e aumentar visibilidade e capacidade de resposta da familia dentro da propria casa, sem expor o produto na internet por padrao.

## Modelo de privilegios

- `control-plane` roda com privilegios minimos.
- Collectors podem receber capacidades extras apenas quando a fonte de rede exigir.
- Containers usam `no-new-privileges` por padrao.

## Dados e retencao

- A base guarda metadados e resumos por padrao.
- Payload bruto continuo nao faz parte da operacao normal.
- Dados sensiveis de trafego nao devem ser persistidos sem habilitacao explicita e temporaria.
- O produto deve preferir evidencias suficientes para decisao, nao coleta excessiva sem necessidade.

## Limites de observacao

- Sem controle do gateway da rede, a visibilidade e parcial.
- `HTTPS` limita inspecao de conteudo.
- O sistema detecta sinais de risco, exposicao e bypass; ele nao substitui um firewall inline completo.
- A cobertura do produto depende das fontes integradas: descoberta, DNS, fluxo e endpoint.
- No modo atual, ele deve ser tratado como ferramenta de observacao e alerta, nao como mecanismo automatico de bloqueio de toda a casa.

## Ameacas cobertas pela base

- dispositivo novo ou nao identificado
- uso de resolvedor DNS fora do esperado
- comunicacao suspeita de camera/IoT
- historico de consultas DNS e fluxo por dispositivo quando a fonte existir
- cameras, baba eletronica e IoT com classificacao suspeita ou exposicao potencial

## Higiene operacional

- revisar periodicamente os perfis e fixtures de integracao
- manter `PostgreSQL`, `NATS`, `AdGuard Home` e collectors atualizados
- usar segmentacao futura para cameras, IoT e dispositivos infantis
