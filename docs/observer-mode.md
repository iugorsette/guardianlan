# Observer Mode

## Resumo

`Observer Mode` e o modo principal do `Guardian LAN` hoje.

Ele foi pensado para funcionar `sem mexer na topologia da rede` e sem exigir que a familia transforme um mini PC em gateway logo no primeiro passo.

## O que ele faz bem

- descobre dispositivos reais da LAN
- identifica fabricante, tipo e risco
- ajuda a reconhecer cameras, tablets, TVs, roteadores e IoT
- mostra mudancas na rede da casa
- gera alertas e explicacoes em linguagem simples
- apoia investigacao de seguranca domestica

## O que ele nao promete

- bloqueio automatico de qualquer celular que entrar no Wi-Fi
- visibilidade total do trafego da casa
- leitura garantida de tudo que qualquer app acessa
- enforcement de rede inteira sem mudanca topologica

## Fontes de observacao possiveis

O `Guardian LAN` consegue alertar melhor quando tem uma fonte real do evento. As principais sao:

- descoberta local da LAN
- logs de DNS, como `AdGuard Home`
- logs ou APIs do roteador, quando existirem
- agente em PCs e notebooks
- export de fluxo, como `ntopng`, NetFlow ou sFlow

## Niveis praticos do produto

### Observer Base

- zero mudanca na rede
- inventario
- descoberta
- classificacao
- alertas de risco
- auditoria de camera e IoT

### Observer Plus

- agente em PCs e notebooks
- DNS opcional em dispositivos escolhidos
- telemetria de dominios de interesse
- watchlists por dispositivo

## Alertas de dominio

Alertas como `xvideos.com` sao possiveis em `Observer Mode`, mas dependem de uma fonte de DNS ou endpoint.

Na pratica, isso funciona quando o sistema recebe o evento por:

- este computador
- notebooks ou desktops com agente
- dispositivos que usem o DNS observado pelo `Guardian LAN`
- integracoes de roteador ou logs que exponham essa informacao

## Posicionamento honesto

O `Guardian LAN` em `Observer Mode` e uma ferramenta de:

- visibilidade
- explicacao
- alerta
- apoio a pais e cuidadores

Ele nao deve ser vendido como firewall invisivel da casa inteira nesse modo.
