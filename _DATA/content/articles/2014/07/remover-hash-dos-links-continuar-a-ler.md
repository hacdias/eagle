---
description: Por padrão, nos artigos do WordPress, quando se clica em "Ler mais", o utilizador é redirecionado para onde estava a ler. Aprenda a remover essa opção.
publishDate: "2014-07-14T13:36:38.000Z"
tags:
- php
title: Remover hash dos links "Continuar a Ler"
---

O **WordPress** é um dos CMS (Sistema de Gestão de Conteúdo, em português) mais conhecidos do mundo, porque é muito simples de utilizar e tem milhares de temas e *plugins* gratuitos e pagos à disposição do utilizador.

Por vezes precisamos de melhorar os temas ou alguma outra coisa. Quando estava a desenvolver o tema deste blog deparei-me com um simples problema: os links dos botões "Continuar a Ler" levavam o utilizador ao local onde havia colocado o código, mas eu não queria que isso acontecesse.

<!--more-->

Depois de uma rápida pesquisa deparei-me com um fórum onde explicavam, em Inglês, como proceder. Afinal é bastante simples. No ficheiro `functions.php` do vosso tema basta colarem o seguinte código:


```php
<?php

function remove_more_link_scroll( $link ) {
  $link = preg_replace( '|#more-[0-9]+|', '', $link );
  return $link;
}

add_filter( 'the_content_more_link', 'remove_more_link_scroll' );
```


Sim, está bem, mas o que faz realmente esse código? Vamos começar por analisar o que existe dentro da função `remove_more_link_scroll()`. Esta aceita uma variável a que vamos chamar `$link`.


## Explicação


Assim, vamos começar pela função `preg_replace()`. Esta aceita três argumentos sendo o primeiro uma expressão regular, o segundo o substituto e o terceiro o texto. Esta função pesquisa no texto a expressão regular e substitui-a.

Neste caso, a função `preg_replace()` substitui no `$link`  todos os caracteres possíveis na expressão `#more-[0-9]+` por '' ou seja, nada. Depois disso, a função retorna o link sem a hash que direciona para a âncora.

Dentro da expressão regular que está contida na função, temos vários "locais" a destacar. Os caracteres que se encontram no início e no final (`|`) servem para delimitar o início e o fim da expressão de forma a que o PHP a consiga detetar.

O texto #more-  corresponde ao início da hash do link. O WordPress faz as âncoras de ler mais iniciando por `#more-`.

O resto, `[0-9]+`, quer dizer que procura por qualquer número entre 0 e 9 e fá-lo mais do que uma vez.

Depois desta função, nós temos que a adicionar aos filtros do WordPress e, para isso, utilizamos as funções próprias do WordPress que, neste caso é a `add_filter()`. Esta aceita quatro argumentos mas só vamos falar de 2 que são os únicos obrigatórios. Se quiseres saber mais sobre a adição de filtros aos temas no WordPress podes aceder a [esta página](http://codex.wordpress.org/Function_Reference/add_filter) do site do WP em inglês.

O primeiro é a tag que, neste caso é 'the_content_more_link' . Podes obter mais informações sobre estas tags [aqui](http://codex.wordpress.org/Plugin_API/Filter_Reference). O segundo argumento é a função a adicionar, que é a função que criámos antes.



* * *



Sei que é uma dica simples mas de certeza que vai ajudar alguém tal como me ajudou a mim :)