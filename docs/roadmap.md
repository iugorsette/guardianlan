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

- conectores reais para roteadores suportados
- ingestao direta do AdGuard Home
- integracao de export de fluxo com ntopng
- agentes de endpoint para notebooks e PCs
- UX guiada para pais e cuidadores
- alertas mais explicativos para camera, tablet infantil, IoT e bypass

## V3

- topologia gateway central ou bridge
- enforcement mais forte por VLAN/segmentacao
- inspeção mais rica de trafego e maior cobertura da rede
- automacoes mais fortes para horarios, perfis e isolamento de risco
