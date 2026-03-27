# Roadmap

O roadmap tecnico acompanha um roadmap de produto: primeiro ajudar pais e cuidadores a enxergar a rede da casa, depois orientar protecao e, por fim, aumentar cobertura quando a topologia permitir.

## V1

- pipeline local com `Rust collectors + Go control plane`
- eventos via `NATS`
- inventario, alertas e API local
- descoberta real da LAN com classificacao inicial de dispositivos
- dashboard local para operacao domestica
- linguagem orientada a seguranca da casa e protecao infantil

## V2

- consolidar o `Observer Mode` como proposta principal do produto
- conectores reais para roteadores suportados em modo observacional
- ingestao direta de logs DNS e fluxo quando existirem
- agentes de endpoint para notebooks e PCs
- UX guiada para pais e cuidadores
- alertas mais explicativos para camera, tablet infantil, IoT, dominio sensivel e bypass
- separar claramente `Observer Base` e `Observer Plus`

## V3

- pesquisas de `appliance mode` e `gateway mode` como trilha opcional futura
- mini PC dedicado apenas quando houver interesse em enforcement real
- conectores reais para roteadores suportados
- integracao mais rica de export de fluxo com ntopng
- inspeção mais rica de trafego e maior cobertura da rede
- automacoes mais fortes para horarios, perfis e isolamento de risco
