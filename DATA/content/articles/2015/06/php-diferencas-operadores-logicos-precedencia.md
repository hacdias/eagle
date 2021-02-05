---
description: Depois de publicar um artigo sobre as diferenças entre os operadores '==' e '===', vou falar sobre as diferenças entre os operadores '&&/||' e 'AND/OR'.
publishDate: "2014-07-28T11:00:32.000Z"
tags:
- php
title: PHP - Diferenças entre '&&/||' e 'AND/OR'
updateDate: "2015-06-10T15:01:00.000Z"
---

Há alguns dias publiquei um artigo onde expliquei as [diferenças entre os operadores `==` e `===`](/articles/2014/07/diferencas-entre-igual-identico-php/). Depois de partilhar esse artigo no Google Plus, a comunidade PHP Brasil sugeriu-me que falasse sobre as diferenças entre os operadores `&&`/`||` e `AND`/`OR`.

Depois de ter pesquisado sobre estes quatro operadores, descobri que sim, realmente **existem diferenças entre** esses dois conjuntos de operadores.

<!--more-->

Para exemplificar, vamos utilizar valores **booleanos**, ou seja, `false` e `true` (ou 0 e 1, respetivamente), e os operadores `AND` e `&&`.

Em primeiro lugar, vamos declarar duas variáveis, uma que seja igual a true  e outra igual a false como as seguintes:

```php
<?php /* ... */

$verdadeiro = true;
$falso = false;
```

Antes de continuarmos, vou recordar algo parecido às regras da multiplicação/divisão com sinais diferentes, mas aqui para verdadeiros e falsos:

|       | **true**  | **false** |
|-------|-------|-------|
| **true**  | true  | false |
| **false** | false | true  |

Esta tabela mostra o resultado de comparações feitas entre os vários valores booleanos. Continuando agora com o "nosso" PHP, vamos definir uma variável chamada, por exemplo, $comparacao e igualá-la a comparações entre as variáveis booleanas anteriormente definidas.

```php
<?php /* ... */

$comparacao = $verdadeiro && $falso;
```

Assim, concluímos que a variável `$comparacao`  é igual a falso. Vamos agora utilizar o operador AND.

```php
<?php /* ... */

$comparacao = $verdadeiro AND $falso;
```

Olhando para a tabela e para a igualdade, pensamos logo que $comparacao  é igual a falso mas... está **errado!** `$comparacao`  é agora igual a **verdadeiro**. Mas como assim?

Isto acontece devido ao **[nível de precedência](http://php.net/manual/pt_BR/language.operators.precedence.php)** dos operadores, ou seja, quando existe mais do que um operador num comando, existem níveis de prioridade sendo uns executados antes do que outros.

As duas linhas acima são equivalentes às seguintes, respetivamente:

```php
<?php /* ... */

$comparacao = ( $verdadeiro && $falso );
( $comparacao = $verdadeiro ) AND $falso;
```

O nível de precedência do operador `=`  é mais elevado que o do operador `AND`. Porém, o nível de precedência do operador `&&` é mais elevado que o de `=`.

Podemos exemplificar isto dos níveis de precedência utilizando matemática. Se escrevermos `5 + 2 * 4`, automaticamente sabemos que vamos efetuar primeiro a multiplicação e só depois a adição, resultando em 13 ou seja, a multiplicação tem prioridade em relação à adição. Assim, `5 + 2 * 4 = 5 + (2 * 4)`.

Só colocando parênteses é que podemos efetuar primeiro a adição e depois a multiplicação: (5 + 2) * 4 . Esta última já resulta em 28.

O mesmo acontece com os operadores `OR` e `||` sendo que o segundo tem um nível de precedência mais elevado que o primeiro. Acrescento ainda que esta explicação é válida para outras linguagens de programação, como Ruby, por exemplo.