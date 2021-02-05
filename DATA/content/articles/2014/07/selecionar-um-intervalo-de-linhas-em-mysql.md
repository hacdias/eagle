---
description: Utilizar MySQL é muito simples. Neste tutorial ensino a selecionar apenas um intervalo de linhas em MySQL.
publishDate: "2014-07-16T20:00:19.000Z"
tags:
- mysql
title: Selecionar um intervalo de linhas em MySQL
---

O sistema de base de dados **MySQL** é dos mais conhecidos mundialmente devido à sua facilidade, tanto de utilização como de implantação e ao facto de ser extremamente versátil.

Quando estava a listar as linhas de uma tabela MySQL numa página web, deparei-me com um sobrecarregamento do CPU do meu computador, porque eu estava a listar **todas** as linhas da tabela. De seguida pensei: **porque não listo por páginas, cada página com `x` linhas?**

<!--more-->

Utilizando este método, o processador ficaria menos carregado de processos, o que aumentaria a experiência do utilizador visto que este, além de receber uma resposta mais rápida, iria ver menos linhas numa página.

## LIMIT e OFFSET

Para limitar o número de resultados dados por uma consulta SQL num sistema MySQL a um intervalo de linhas, os comandos  LIMIT e OFFSET devem ser utilizados. Pode ver a consulta (*query*) abaixo para um exemplo ilustrativo.

```sql
SELECT * FROM `mytable` LIMIT 15
```

O resultado da consulta acima serão as primeiras 15 linhas da tabela `mytable`. Mas, e se eu quiser as segundas 15 linhas e não as primeiras?

Para o fazermos, apenas temos que utilizar o comando  OFFSET . Então, para obtermos as segundas quinze linhas de uma tabela MySQL, faríamos o seguinte:

```sql
SELECT * FROM `mytable` LIMIT 15 OFFSET 15
```

Em Português, *offset* quer dizer deslocamento. Podemos então dizer que a *query* está a ser deslocada para outro local da tabeça passando à frente das primeira 15 linhas, selecionando as segundas 15.

Poderíamos ainda simplificar o código acima para o seguinte:

```sql
SELECT * FROM `mytable` LIMIT 15,15
```

Estes comandos podem ser utilizados com outros como o WHERE .

Em breve irei lançar um tutorial que explique como criar uma listagem de uma tabela, com páginas utilizando PHP, HTML e MySQL. Este sistema irá contar com um número fixo de linhas por página, botões de navegação e forma de impedir que o utilizador tente aceder páginas inexistentes.

Espero que esta explicação vos tenha sido útil que voltem a visitar o blog :)