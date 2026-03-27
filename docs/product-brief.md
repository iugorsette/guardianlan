# Product Brief

## Tese

`Guardian LAN` e uma plataforma `local-first` para observar, explicar e alertar sobre riscos digitais na rede domestica sem depender de nuvem como requisito central.

O produto nasce para ajudar pais, maes e cuidadores que nao dominam tecnologia, mas precisam:

- entender o que existe na rede da casa
- descobrir cameras, tablets, celulares, TVs, consoles e IoT
- receber alertas simples sobre risco real
- ajudar pais a enxergar sinais digitais sem virar especialista em firewall

## Problema que o produto resolve

Na pratica, protecao digital domestica costuma ser:

- cara
- fragmentada entre varios apps e servicos
- tecnica demais para familias leigas
- dependente de cloud e contas externas
- ruim para explicar riscos de uma camera, baba eletronica ou tablet infantil

O `Guardian LAN` tenta reduzir esse atrito com uma experiencia local, explicativa e progressiva.

## Publico principal

- pais e maes com baixa familiaridade tecnica
- familias com bebe, crianca ou adolescente
- casas com baba eletronica, camera IP, smart TV, tablet, console e dispositivos IoT
- pessoas que querem mais seguranca local sem colocar tudo na internet

## Principios do produto

- `local-first`
- `privacy-aware`
- `linguagem simples`
- `seguranca por padrao`
- `baixo custo operacional`
- `explicacao orientada a acao`

## Casos de uso centrais

- descobrir quem entrou ou apareceu na rede da casa
- identificar dispositivos que parecem cameras, roteadores, TVs ou IoT
- ajudar pais a organizar perfis como `crianca`, `adolescente`, `iot` e `visitante`
- mostrar risco de forma compreensivel
- reduzir exposicao indevida de cameras e baba eletronica
- servir como base domestica de observabilidade e supervisao parental

## Nao objetivos da v1

- nao prometer visibilidade total de todo o trafego da rede
- nao depender de MITM generalizado para funcionar
- nao exigir troca imediata do roteador
- nao operar como produto SaaS obrigatorio
- nao prometer bloqueio automatico de toda a casa sem topologia adequada

## Contexto regulatorio

O projeto conversa com o movimento de protecao de criancas e adolescentes em ambientes digitais no Brasil, inclusive com a `Lei nº 15.211/2025`, conhecida como `ECA Digital`, mas a proposta aqui e domestica e local:

- proteger a casa
- orientar pais e cuidadores
- reduzir barreiras tecnicas e de custo

## Proposta de UX

O sistema deve falar mais como um assistente domestico de seguranca do que como uma ferramenta de SOC. Exemplos:

- `Encontramos uma camera possivelmente ativa na sua rede`
- `Este dispositivo esta usando a internet fora da protecao esperada`
- `Este tablet parece ser de uso infantil; deseja aplicar perfil Crianca?`
- `Uma baba eletronica pode estar exposta; revise este equipamento`

## Direcao de deploy

Hoje, o produto deve ser descrito primeiro como `Observer Mode`:

- roda em uma maquina da casa
- observa a rede local sem exigir mudar a topologia
- usa fontes opcionais de DNS, fluxo e endpoint quando existirem
- prioriza inventario, explicacao e alerta

Pesquisas de `mini PC`, `appliance mode` e `gateway mode` continuam existindo no repositório, mas ficam como direcao futura e nao como promessa principal do produto atual.
