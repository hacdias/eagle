---
description: Já pensou em qual a melhor forma para inserir variáveis dentro de strings? A interpolação ou a concatenação. Venha ver.
publishDate: "2014-09-09T15:19:43.000Z"
tags:
- php
title: 'PHP - Strings: interpolação e concatenação'
---

Hoje vamos falar um pouco sobre PHP, mais precisamente no campo das strings, variáveis e concatenações. Como sabem, existem várias formas de inserir o valor de variáveis dentro de strings, mas qual será a "melhor" e mais rápida?

<!--more-->

Em primeiro lugar, vamos rever as formas de inserir variáveis dentro de _strings_ atualmente já existentes:

```php
<?php
$foo = 'uma pessoa';

$bar = "Eu sou " . $foo . "!"; // => 1
$bar = 'Eu sou ' . $foo . '!'; // => 2
$bar = "Eu sou $foo!"; // => 3
$bar = "Eu sou {$foo}!"; // => 4
```

Vamos analisar os quatro exemplos acima sendo que os primeiros dois é utilizada **concatenação** e nos últimos dois **interpolação**.

## Métodos 1 e 2

Vamos começar por analisar o primeiro. Neste caso, o segundo método devia ser utilizado ao invés do primeiro. Porquê?

As aspas duplas dizem ao PHP para iniciar a interpolação gastando mais recursos e podendo demorar mais tempo. Devemos utilizar a aspa única quando não utilizamos nenhum benefício da interpolação como `n`, por exemplo.

## Métodos 3 e 4

Em relação ao terceiro e quarto, é indiferente porém o uso de chavetas é extremamente recomendado quando se inserem variáveis mais complexas como _arrays_.

Caso sejam variáveis simples, o uso de chavetas é desnecessário não trazendo benefícios nem malefícios.

## Qual devo usar?

Depende! Geralmente, a interpolação é mais lenta porém, a concatenação pode tornar-se mais lenta caso sejam utilizadas muitas variáveis.

Executei seguinte teste para confirmar as velocidades que cada um demora a correr (função [timeFunc](http://stackoverflow.com/questions/13620/speed-difference-in-using-inline-strings-vs-concatenation-in-php5) obtida aqui):

```php
<?php

function timeFunc($function, $runs)
{
  $times = array();

  for ($i = 0; $i < $runs; $i++)
  {
    $time = microtime();
    call_user_func($function);
    $times[$i] = microtime() - $time;
  }

  return array_sum($times) / $runs;
}

function Method1()
{
  $foo = 'uma pessoa';
  for ($i = 0; $i < 10000; $i++)
    $t = "Eu sou " . $foo . "!"; //1
}

function Method2()
{
  $foo = 'uma pessoa';
  for ($i = 0; $i < 10000; $i++)
    $t = 'Eu sou ' . $foo . '!'; //2
}

function Method3()
 {
  $foo = 'uma pessoa';
  for ($i = 0; $i < 10000; $i++)
    $t = "Eu sou $foo!"; //3
}
function Method4()
 {
  $foo = 'uma pessoa';
  for ($i = 0; $i < 10000; $i++)
    $t = "Eu sou {$foo}!"; //4
}

echo timeFunc('Method1', 10) . "n"; // => 0.0020885
echo timeFunc('Method2', 10) . "n"; // => 0.0021168
echo timeFunc('Method3', 10) . "n"; // => 0.0021132
echo timeFunc('Method4', 10) . "n"; // => 0.0023884
```

Recebi os valores mencionados nos comentários. Como podem ver, não existem grandes diferenças no tempo de execução destes pequenos exemplos. Espero que o post tenha sido útil.