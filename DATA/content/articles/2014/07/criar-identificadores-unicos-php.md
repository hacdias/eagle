---
description: Como você sabe, PHP é das linguagens de servidor mais utilizadas no mundo. Hoje trago-vos uma das funções mais interessantes desta linguagem.
publishDate: "2014-07-30T08:34:52.000Z"
tags:
- pgp
title: Criar identificadores únicos em PHP"
---

Como deve saber, PHP é umas das linguagens de servidor mais utilizadas no mundo. Hoje vou mostrar-vos uma fantástica forma de gerar **IDs (identificadores) únicos**.

<!--more-->

O método para o fazer chama-se **uniqid()** e deve ser utilizado da seguinte forma.

```php
<?php /*...*/

string uniqid ([ string $prefix = "" [, bool $more_entropy = false ]] )
```

Esta função cria um ID baseado nas horas atuais em microssegundos logo não são criados números aleatórios. Se chamar esta função sem quaisquer parâmetros, o PHP irá fornecer-lhe um conjunto de 13 caracteres. Exemplo:

```php
<?php /*...*/

for ($i = 0; $i < 10; $i++) {
  echo uniqid() . '<br>';
}
```

E eu recebi o seguinte:

```txt
540f25baa463c
540f25baa4644
540f25baa4647
540f25baa4649
540f25baa464c
540f25baa464f
540f25baa4652
540f25baa4654
540f25baa4657
540f25baa4659
```

Como pode ver, existe um padrão que é seguido em cada entrada. Agora, vamos tornar isto mais interessante com prefixos. O primeiro parâmetro pode aer utilizado para prefixos. Exemplo:

```php
<?php /*...*/

for ($i = 0; $i < 10; $i++) {
  echo uniqid('id.') . '<br>';
}

// =>
// id.53d8a6b00ab54
// id.53d8a6b00ab61
// id.53d8a6b00ab65
// id.53d8a6b00ab6a
// id.53d8a6b00ab6e
// id.53d8a6b00ab73
// id.53d8a6b00ab78
// id.53d8a6b00ab7d
// id.53d8a6b00ab80
// id.53d8a6b00ab85
```

Antes de continuarmos para o segundo parâmetro vamos tornar isto mais divertido utilizando a função ```rand()``` para os prefixos como seguinte:

```php
<?php /*...*/

for ($i = 0; $i < 10; $i++) {
  echo uniqid(rand()) . '<br>';
}

// =>
// 1847953d8a7a793dad
// 1593253d8a7a793dc6
// 3066153d8a7a793dd3
// 1359453d8a7a793de0
// 1441153d8a7a793ded
// 55253d8a7a793df9
// 1926553d8a7a793e05
// 227853d8a7a793e11
// 362353d8a7a793e1d
// 635653d8a7a793e2a
```

Pode aprender mais sobre esta função aqui [aqui](http://pt2.php.net/manual/en/function.rand.php). Podemos também ativar a entropia utilizando o segundo parâmetro.

```php
<?php /*...*/

for ($i = 0; $i < 10; $i++) {
  echo uniqid(NULL,  true) . '<br>';
}

// =>
// 53d8a804e20009.43950771
// 53d8a804e201f6.55814947
// 53d8a804e20303.56124572
// 53d8a804e203f8.58127959
// 53d8a804e204e1.29334755
// 53d8a804e205d8.15855084
// 53d8a804e206c1.12343150
// 53d8a804e207b1.83476235
// 53d8a804e208b5.92283530
// 53d8a804e209d6.58175306
```

Com entropia activada, o tamanho do conjunto passa de 13 a 23 caracteres. Podemos ainda utilizar a entropia com o comando ```rand()``` ao mesmo tempo:

```txt
654753d8a87287e1c6.37774954
2401653d8a87287e2e3.55006223
45753d8a87287e363.99897593
1249453d8a87287e3d4.10326514
1534353d8a87287e451.40348788
178853d8a87287e4c0.49212408
2906153d8a87287e547.79619666
303453d8a87287e5c2.76725867
1025153d8a87287e649.36971601
586453d8a87287e6c7.26848089
```

Isto é bom para quando se tem mais de um servidor a correr a mesma aplicação. Assim evitam-se possíveis erros causados por IDs iguais. Como assim? Se tiverem dois ou mais servidores a correr a mesma aplicação, pode acontecer algum correr o comando no mesmo micro segundo.

É isto. :)