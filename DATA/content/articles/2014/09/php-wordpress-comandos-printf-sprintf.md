---
description: Existem vários comandos que nos permitem imprimir frases. Porque é que existem vários? Alguns tornam o texto mais claro, como o 'printf' e o 'sprint'.
publishDate: "2014-09-13T09:16:48.000Z"
tags:
- php
title: 'PHP e WordPress: comandos ''printf'' e ''sprintf'''
---

No [último artigo](/articles/2014/09/php-interpolacao-concatenacao/) sugeriram-me falar sobre as funções printf  e  sprintf  que são utilizadas em massa no Wordpress e é isso que vou fazer! Vamos analisar cada uma das funções.

Vou começar por analisar as duas funções em separado, explicando para que servem e dando alguns exemplos.

<!--more-->

## printf

O nome desta função quer dizer _print formatted_, ou seja, "imprimir dados formatados". Abaixo encontra um exemplo mais simples:

```php
printf("Olá mundo!");

//Olá mundo!
```

E agora você pergunta-me: que utilidade tem essa função se podemos fazer o mesmo com `echo` ou `print`?

É aqui que está função se destaca. Compare as seguintes impressões, todas vão imprimir o mesmo.

```php
<?php

$foo = "Henrique";
$bar = "14";
$site = "COXPE";

//Chamo-me Henrique, tenho 14 anos e estou a navegar no COXPE.

echo 'Chamo-me ' . $foo . ', tenho ' . $bar . ' anos e estou a navegar no ' . $site . '.';

echo "Chamo-me {$foo} tenho {$bar} anos e estou a navegar no {$site}.";

printf("Chamo-me %s e tenho %d anos e estou a navegar no %s.",
	$foo, $bar, $site);
```

Se repararmos, das três, a última é a que tem uma maior legibilidade, tal como o leitor Carlos Santos tinha mencionado.

Como pode ver, existe ali um `%s` e um` %d` que são substituídos pelas variáveis que coloco depois. Existem vários "por centos":

  * `%` → imprime um sinal "%"
  * `%b` → permite o envio de um número inteiro que será imprimido em binário
  * `%c` → permite o envio de um número inteiro e será imprimido o caractere ASCII correspondente.
  * `%d` → permite o envio de um número inteiro e imprime-o
  * `%e` → o argumento é tratado como notação científica
  * `%E` → igual ao anterior porém o "e" tem que ser maiúsculo ("12E5" em vez de "12e5")
  * `%f` → o argumento é tratado como _float_
  * `%g` → atalho para `%e` e `%f`
  * `%G` → atalho para `%E` e `%f`

Estes são os mais utilizados porém podem ver mais [aqui](http://php.net/manual/en/function.sprintf.php). Assim, para utilizar esta função, seguimos a seguinte sintaxe:

```php
printf($formato[, $restantesArgumentos...])
```

O $formato  é a string que contém a frase a ser imprimida com as diversas diretivas que podem ser 0 ou mais. É indiferente.

Os restantes argumentos são os dados que são para ser enviados para essas diretivas por ordem de aparecimento na string.

## sprintf

A única diferença entre esta função e a `printf`  é que esta **retorna** a string formatada e `printf`  **imprime** a string formatada.

## WordPress, traduções e estas funções...

O WordPress usa estas funções em massa juntamente com o sistema de tradução tornando esta função extremamente potente.

Este CMS usa o sistema Gettext para traduzir o conteúdo que nos é visível. Agora não vamos entrar em detalhes sobre a ativação deste sistema (se tiverem qualquer dúvida coloquem).

O WordPress deve utilizar este sistema também pela simplicidade de leitura visto que os comandos de tradução do Gettext são um quanto "complicados" de ler.

```php
<?php

//Exemplo de "printf" utilizado pelo WordPress
printf( __( 'Ready to publish your first post? <a href="%1$s">Get started here</a>.', 'twentyfourteen' ), admin_url( 'post-new.php' ) );

//Em Echo
$url = admin_url( 'post-new.php' );
echo __( "Ready to publish your first post? <a href="{$url}">Get started here</a>.", 'twentyfourteen' );
```

A forma que coloquei com echo poderia variar. Mas, como pode visualizar, o printf é mais simples de ler.

Vou deixar esta parte para um outro artigo ;) Deixo já nos rascunhos. Em breve poderei fazer um artigo sobre o Gettext, como ativar e usar! =D

* * *

**_A partir de segunda-feira, a frequência de lançamento de artigos irá diminuir porque a escola vai recomeçar. Continuarei a lançar a [rubrica semanal](http://pplware.sapo.pt/tutoriais/programacao/vamos-programar-introducao-a-programacao-2/) com o Pplware e também no [Pplware Kids](http://kids.pplware.sapo.pt/). Isto não quer dizer que deixarei de escrever para o COXPE. Sempre que puder, virei aqui :) Obrigado pela compreensão._**