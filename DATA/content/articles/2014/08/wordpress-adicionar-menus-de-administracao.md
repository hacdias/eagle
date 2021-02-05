---
description: A criação de temas e plugins para o WordPres leva a uma necessidade constante de criar menus de administração. A criação dos mesmos é muito simples.
publishDate: "2014-08-02T11:27:37.000Z"
tags:
- php
title: 'WordPress: adicionar menus de administração [Parte 1]'
---

O **WordPress** é, como vocês sabem, um dos mais populares CMS do mundo. Isto, se não o for. Um dos nossos primeiros artigos foi referente ao WordPress e ensinámos a [remover a hash dos links "Ler mais"](/articles/2014/07/remover-hash-dos-links-continuar-a-ler/).

Hoje vamos dar continuação às dicas e tutoriais sobre este CMS. Se vocês utilizam o WordPress com um tema pago, já devem ter reparado que este vem com um menu de personalização próprio criado pelo(s) autor(es) do tema. Hoje vamos ensinar-vos a criar estes menus.

<!--more-->

### 1. Criar ficheiro das opções e chamá-lo

_O tutorial é direcionado para **temas** porém a criação dos menus faz-se da mesma forma nos **plugins**._

Começaremos por criar um ficheiro chamado `admin.php` na pasta `inc` do vosso tema (ou plugin). Caso esta pasta não exista, podem criá-la ou colocá-lo numa outra pasta que queiram.

Antes de continuarmos o desenvolvimento deste ficheiro, vamos chamar este ficheiro através do `functions.php`. **Neste, coloquem o seguinte código:**

```php
<?php

//...

require get_template_directory() . '/inc/admin.php';
```

Este pequeno trecho de código chama o ficheiro através do comando `require` e da localização do mesmo. Para obtermos a localização da pasta principal do tema utilizamos a função `get_template_directory()`.

Depois concatenamos o caminho do tema ao resto do directório que, no nosso caso é `/inc/admin.php` . Se colocaram noutra pasta não se esqueçam de alterar o caminho.


### 2. Hora de criar as opções


Agora sim, vamos virar-nos para o ficheiro `admin.php`. Neste ficheiro irá estar todo o código referente às opções de administração do tema.

Neste tutorial, iremos criar uma variável com um dado que será depois utilizada no tema. Assim, aprendem como se faz e depois podem criar vocês próprios as opções que quiserem.

Vamos começar por criar a seguinte função:

```php
<?php

function themeslug_admin_menus() {
    //Colocaremos aqui os menus
}

add_action("admin_menu", "themeslug_admin_menus");
```

Não se esqueça de substituir `themeslug` pelo prefixo das funções do seu tema. Criámos uma pequena função que vai servir para adicionar os menus à barra de administração do WordPress.

Depois adicionamos essa função utilizando a função add_action . Este comando liga uma **função** a uma **ação**. Podem ler mais sobre a mesma [aqui](http://codex.wordpress.org/Function_Reference/add_action).

Existem várias formas de adicionar um menu. Podemos adicionar um sub-menu ou um menu. Vou dar-vos um exemplo com algo já existente no WP:

{{< figure src="https://cdn.hacdias.com/uploads/2014-08-wpsubmenu.jpeg" alt="WordPress Menu e Sub-menu" >}}

  * **Settings** é um menu.
  * **Media** é um sub-menu do menu **Settings**.

Para adicionar um sub-menu a um menu, utilizamos a função `add_theme_page` que tem a seguinte sintaxe:

```php
add_theme_page( $titulo_da_pagina, $titulo_do_menu, $permissoes, $slug_do_menu, $funcao);
```

Vamos por partes:

  * `$titulo_da_pagina`  -> Obviamente que é o título da página;
  * `$titulo_do_menu`  -> Nome que irá aparecer na barra de administração;
  * `$permissoes`  -> O mínimo de permissões que o utilizador deve ter para aceder à página;
  * `$slug_do_menu`  -> O nome do menu sem caracteres especiais e sem maiúsculas para ser utilizado no URL;
  * `$funcao`  -> A função que vai "tratar" deste sub-menu.

Para criar um menu de nível principal (como *Settings*), utiliza-se a seguinte sintaxe:

```php
<?php

add_menu_page( $titulo_da_pag, $titulo_do_menu, $permissoes, $slug_do_menu, $funcao, $icon_url, $posicao );
```

A sintaxe é muito parecida mas conta com mais duas variáveis. A `$icon_url` serve para indicar o caminho do ícone e $posicao  é a posição na barra lateral da *dashboard*. Ambas são opções opcionais e podes ler mais sobre elas [aqui](http://codex.wordpress.org/Function_Reference/add_menu_page).

Vamos, finalmente, criar o menu. Já conhecendo a sintaxe, vou substituir o comentário que escrevemos no ficheiro `admin.php` pela criação de um sub-menu ficando o código da seguinte forma:

```php
<?php

function themeslug_admin_menus() {

    //Adição do sub-menu Opções ao menu principal Apresentação
    add_theme_page('Opções', 'Opções', 'manage_options', 'opcoes', 'themeslug_options');
}

function themeslug_options() {
    //Esta função vai conter a página "Opções".
}

add_action("admin_menu", "themeslug_admin_menus");
```

Agora, quando voltarem à *dashboard* do WordPress, já lá deverão ter o novo menu.

Se abrirem a página vão ver que ainda não existe nada nessa página. Como este tutorial está a ficar um pouco extenso, deixarei a criação de uma opção e a sua aplicação para a segunda parte deste tutorial que <del>deve chegar dentro de dois ou três dias</del> podes encontrar [aqui](/articles/2015/06/wordpress-adicionar-menus-de-administracao-parte-2/).

Esperamos que o tutorial vos tenha sido útil.