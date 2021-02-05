---
description: A tag de fechamento do PHP ( ?> ) pode causar alguma confusão visto que é omitida pela maioria dos desenvolvedores. Mas porquê?
publishDate: "2014-08-27T09:15:18.000Z"
tags:
- php
title: PHP - Devemos usar a tag de fechamento ou não?
---

Recentemente comecei a reparar que muitos programadores omitiam a tag de fechamento dos ficheiros PHP e, obviamente, fiquei curioso.

Depois de uma pesquisa, trago-vos o **porquê** de não utilizar a tag `?>` no final dos ficheiros.

<!--more-->

Em primeiro lugar, esta prática só deve ser realizada em ficheiros cujo seu conteúdo seja **somente PHP** e não contenha HTML, por exemplo.

## O que acontece se...?

Vamos começar por debater a pergunta que vocês vêm aí em cima: **o que acontece** se omitirmos a *tag* de fechamento.

A resposta é muito simples: se omitirmos a *tag* de fechamento de PHP, este irá tratar todos os espaços vazios e quebras de linha como caracteres "inúteis".

Agora, invertendo a pergunta, **o que acontece se utilizarmos a tag de fechamento**?

Se o fizermos, tudo o que se encontra depois dessa tag irá ser enviado para o browser e, além disso poderá haver problemas com os cabeçalhos HTTP.


## Como assim?


Existem diversas funções que utilizamos frequentemente no código como `session_start()`, `header()`, dentro de muitas outras que alteram estes cabeçalhos.

Ou seja, se alguém cometer qualquer descuido e colocar, simplesmente, diversas linhas em branco no ficheiro, poderá ocorrer um erro ao utilizar funções que utilizem os cabeçalhos.


## Assim...


Depois de eu próprio ter lido tudo isto descobri a solução a um grande problema que estava a ter:


> Cannot modify header information – headers already sent


Este erro leva-nos, por vezes, a grandes "buscas" sendo o problema simplesmente simples. E que tal, já sabiam desta pequena "dica"?

* * *

Aqui estão alguns exemplos para suportar o que disse acima. Espero que sejam úteis.

## Somente PHP

Caso o ficheiro apenas contenha código PHP, não necessitamos de fechar o código com a tag ?>.

```php
<?php

$myString = 'MyString';
$myArr = [0, 'item', $myString];

function my_function {
  //...
}
```

## Utilização de PHP e HTML

Estes primeiros exemplos exemplificam a forma como devemos utilizar as tags em ficheiros que contenham tanto PHP como HTML.

### PHP Primeiro

Se colocarmos o código PHP primeiro, *temos* que utilizar a tag de fechamento do PHP (?>). Caso contrário será gerado um erro.

```php
<?php

//...Código PHP

?>

<html>
  <head>
  
  </head>
  
  <body>
  
  </body>
  
</html>
```

### PHP intercalado

Se utilizarmos o PHP intercalado no HTML devemos também utilizar a tag de fechamento do PHP. Se utilizarmos PHP *depois* de termos fechado a tag principal do código HTML, *não precisamos de utilizar a tag ?>*. 

```php
<html>
  <head>
  
  </head>
  
  <body>
    <div>
      <?php
      
      //...
  
      ?>
    </div>
    <p> <?php echo $myVar; ?> </p>
  </body>
  
</html>

<?php

//...
```

### Cuidaddo com os cabeçalhos

Se utilizarmos PHP e HTML juntos no mesmo ficheiro, devemos nos certificar de que todas as funções que utilizem os cabeçalhos HTTP são colocadas *antes* de qualquer output HTML.

```php
<?php

session_start();

?>

<html>
  <!-- ... Não será gerado nenhum erro  -->
</html>
```

### 'Include' e 'Require'

Perguntaram-me o seguinte:

> (...) se eu incluir o arquivo PHP sem o fechamento em um arquivo que contenha HTML não ocorre erro?﻿

Ou seja, perguntaram-me se pode ocorrer algum erro numa situação semelhante à seguinte:

#### class.php

```PHP
<?php

$myInt = 5;
$myString = 'Yeah! I am on GitHub!';

function MyFunc() {
  //...
}

//...
```

#### index.php

```php
<?php 

include 'class.php'; //Ou require

?>

<html>
  <head>
    <title>Exemplo</title>
  </head>
  
  <body>
    <!-- Tarararara... -->
  </body>
</html>
```

A resposta é *não* mas **tudo depende**. Não se esqueçam de ter cuidado com os cabeçalhos HTTP.