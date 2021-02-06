---
description: O PHP 5.6.0 trouxe aos desenvolvedores novas funcionalidades e formas de trabalhar. Uma das melhores novidades é a nova sintaxe das funções variádicas.
publishDate: "2014-08-31T09:37:05.000Z"
tags:
- php
title: PHP 5.6 - Funções Variádicas (Nova Sintaxe)
---

Há pouco mais de dois dias, foi lançada uma nova versão do PHP, [a versão 5.6.0](http://php.net/archive/2014.php#id2014-08-28-1) que trouxe inúmeras novidades e incompatibilidades com as versões anteriores [(ver aqui)](https://www.php.net/manual/pt_BR/migration56.php).

Hoje vamos falar de uma das diversas grandes novidades desta esplêndida atualização, as **funções variádicas**.

<!--more-->

Ao longo das versões 4.x e 5.x do PHP, os desenvolvedores já podiam utilizar funções variádicas através do uso da função ```func_get_args()```.

Se, por exemplo, quiséssemos criar uma função que fizesse a soma de todos os valores passados nos argumentos, bastaria fazermos assim:

```php
function soma() {
  return array_sum(func_get_args());
}

echo soma(1, 4, 12, 20); // => 37
```

Apesar desta forma ainda estar disponível na versão 5.6.0, a sua sintaxe pode ser aprimorada para a seguinte:

```php
function soma(...$nums) {
  return array_sum($nums);
}

echo soma(1, 4, 12, 20); //= 37
```

Ou seja, na nova versão do PHP, basta utilizarmos o operador ...  para criarmos um _array_ que contenha todos os restantes parâmetros enviados.


## Perda de tempo menor


Uma vantagem na utilização deste método é a **menor perda de tempo**. Como assim? Vou utilizar um exemplo dado pelos colegas do [imasters.com.br](http://imasters.com.br). Em vez de escrevermos assim:

```php
public function tryMethod() {

        $args = func_get_args();
        $method = $args[0];
        unset($args[0]);
        $args = array_values($args);

    try {

        return call_user_func_array([$this, $method], $args);

    } catch (Exception $e) {

        return false;

    }
}
```

Podemos, simplesmente, escrever da seguinte forma:

```php
public function tryMethod($method, ...$args) {

    try {

        return call_user_func_array([$this, $method], $args);

    } catch (Exception $e) {

        return false;

    }
}
```


## Legibilidade


Simples, não é? Outra grande vantagem é a **legibilidade** visto que o código fica mais legível utilizando esta sintaxe. O exemplo acima é, também, um bom exemplo para este ponto.


## Conclusão


Falando mais no geral e vendo a lista de [novas funcionalidades/alterações](https://www.php.net/manual/pt_BR/migration56.new-features.php) da nova versão do PHP, podemos verificar que a versão traz muitas melhorias.


> O que acham vocês, leitores e desenvolvedores, desta nova versão do PHP?