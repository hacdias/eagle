---
description: O operador ? é, por vezes, intrigante. Chama-se operador ternário e explicamos para que serve este operador condicional em PHP.
publishDate: "2014-07-22T09:08:08.000Z"
tags:
- php
title: Operador Ternário ( ? ) em PHP
---

Recentemente, escrevi um artigo onde explicava como fazer uma [listagem web de uma tabela MySQL](/articles/2014/07/listagem-web-de-uma-tabela-mysql/) onde utilizei um operador que não tinha a certeza absoluta do que fazia e deixei a explicação um pouco vaga.

O operador em questão era o ponto de interrogação. Depois de uma pequena pesquisa, descobri que o seu nome é **operador ternário** e é um operador condicional.

<!--more-->

Já utilizava este operador há algum tempo mas estava reticente em relação à sua sintaxe pois não tinha a certeza se o que estava a fazer estava 100% correto por isso decidi pesquisar.

Este operador é excelente para pequenas e simples condições de `if else` onde não pretendemos utilizar muitas linhas.

```php
<?php /* ... */

$url = isset($_GET['url']) ? $_GET['url'] : null;
```

Esse excerto acima foi tirado do artigo que mencionei acima e faz o mesmo que o código abaixo:

```php
<?php /* ... */

if (isset($_GET['url']) {
      $url = $_GET['url'];
} else {
      $url = null;
}
```

Vendo isto, concluímos que o  operador ternário é mais simples de utilizar, porém um pouco mais difícil de ler.

A palavra "ternário" provém de "três" e é utilizada neste operador porque este precisa de três argumentos. A sintaxe é a seguinte:

```txt
(CONDIÇÃO)  ? <O QUE FAZ SE FOR VERDADEIRO> : <O QUE FAZ SE FOR FALSO>
```

Abaixo encontra-se mais um pequeno exemplo:

```php
<?php /* ... */

$n = rand(0,100);

if ($n > 50) {
    echo 'O número é maior que 50!';
} else {
    echo 'O número é menor que 50!';
}

//Utilizando o operador ternário ficaria:

echo ($n > 50) ? 'O número é maior que 50!' : 'O número é menor que 50!';

//O leitor Gustavo Rafael sugeriu uma forma mais simplificada:
echo 'O número é ' . (($n > 50) ? 'maior' : 'menor') . ' que 50!';
```

Mais uma vez podemos concluir que utilizando  o operador ternário gastamos menos linhas e poupamos *bytes* no tamanho do ficheiro.

Para saberem mais sobre operadores em PHP podem aceder a [esta página](http://br2.php.net/manual/en/language.operators.comparison.php) no guia oficial da linguagem. Este operador existe também em outras linguagens como C ou JS por exemplo.

Espero que tenham gostado desta pequena explicação. :)