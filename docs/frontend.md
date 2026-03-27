# Frontend

## Escolha atual

O painel foi iniciado em `Angular` por tres motivos principais:

- combina melhor com um produto administrativo que tende a crescer em modulos
- voce ja tem conforto com o ecossistema
- `standalone components + signals` entregam uma base moderna sem perder organizacao

No contexto do produto, a UI nao e apenas um dashboard tecnico. Ela precisa servir como uma interface de orientacao para pais e cuidadores, com linguagem simples e contexto suficiente para decisao.

No modo atual do produto, a UI deve falar como `Observer`:

- mostrar o que foi visto
- explicar por que aquilo importa
- alertar pais e cuidadores
- evitar linguagem que pareca prometer bloqueio automatico quando a topologia nao suporta isso

## Quando Angular faz mais sentido aqui

- muitas telas administrativas
- formularios de politica e configuracao
- listas, tabelas e filtros por dispositivo
- operacao de longo prazo com UX previsivel

## Comparacao curta

- `Angular`
  - melhor para estrutura e evolucao disciplinada
  - custo: mais verbosidade que React/Vue
- `React`
  - mais flexivel e com ecossistema enorme
  - custo: exige mais disciplina arquitetural para nao espalhar estado e padroes
- `Vue`
  - meio termo muito agradavel
  - custo: menor alinhamento com seu conforto atual
- `Svelte`
  - runtime muito enxuto
  - custo: menor ecossistema para paineis administrativos maiores

## Limites reais de performance

- para este produto, o gargalo raramente sera o framework
- os limites vao aparecer antes em polling, tabelas grandes, graficos, atualizacao em tempo real e volume de eventos
- o caminho de performance aqui passa mais por paginacao, agregacao e streaming inteligente do que por trocar Angular por outro framework

## Direcao adotada

- `Angular 21`
- componentes standalone
- `signals` para estado local da tela
- proxy `/api` para integrar com o control plane
- Docker com `nginx` para servir a build estatica em producao local
