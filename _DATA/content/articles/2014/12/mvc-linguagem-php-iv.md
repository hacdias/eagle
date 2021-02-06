---
description: No último tutorial sobre a criação de uma framework que segue o modelo MVC, criamos uma página dinâmica com acesso à base de dados.
publishDate: "2014-12-21T10:00:04.000Z"
tags:
- php
title: MVC na linguagem PHP - IV
---

A pedido de um leitor, decidimos escrever um artigo em que exemplificamos este maravilhoso (ou não) modelo MVC utilizando a linguagem de programação PHP.

A [primeira](/articles/2014/12/mvc-linguagem-php/), [segunda](/articles/2014/12/mvc-linguagem-php-ii/) e [terceiras](/articles/2014/12/mvc-linguagem-php-iii/) partes já foram publicadas. Hoje, trago a quarta e última parte desta mini-série de artigos.

<!--more-->

O que vamos fazer hoje é, simplesmente, criar uma página dinâmica que liste diversos _posts_ que estão na base de dados da aplicação.

Se bem se lembra, a ligação à base de dados é configurada no ficheiro `config.php`, por isso assegure-se que a sua conexão está bem configurada.

Aqui, tenho uma base de dados chamada _mvc_ com uma tabela chamada _posts_, e esta tabela tem três colunas: a `id`, `title` e `content`.

De momento tenho duas linhas inseridas nessa mesma tabela, ou seja, de momento tenho exatamente o seguinte:

| id | title       | content                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
|----|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 1  | Lorem Ipsum | Lorem ipsum dolor sit amet, consectetur adipiscing elit. Phasellus id velit non tellus feugiat feugiat vitae quis nibh. Pellentesque maximus lectus ut enim tincidunt, a rutrum dui elementum. Vestibulum elit sapien, malesuada sit amet est lacinia, aliquet laoreet arcu. Duis quis velit hendrerit, pretium nibh ut, faucibus odio. Fusce hendrerit nunc urna, vitae varius augue fringilla in. Nunc a ex eget lectus dictum mollis. Proin quis nisl consectetur metus bibendum ultricies eu non orci. Quisque nec efficitur quam. Suspendisse lorem nulla, sollicitudin ac sagittis id, eleifend eget eros. Interdum et malesuada fames ac ante ipsum primis in faucibus. Cras neque leo, consectetur nec sem quis, imperdiet viverra quam. Pellentesque ultricies felis a molestie egestas. Aliquam malesuada eget justo condimentum venenatis. Duis mattis ut nisi in suscipit. Phasellus scelerisque, arcu ut sollicitudin sagittis, ex ante posuere neque, et dapibus ex lectus quis libero.                                                    |
| 2  | Usto Risus  | Usto risus, cursus non iaculis a, semper vitae dolor. Nunc pellentesque tempor pretium. Sed sem risus, accumsan ut urna in, sollicitudin sagittis nisi. Integer ullamcorper orci id nisl iaculis, ac congue purus posuere. Vivamus pharetra nibh in arcu vulputate, in feugiat dolor feugiat. Aliquam erat volutpat. Maecenas sodales magna urna, quis faucibus arcu mattis sit amet. Ali. Fusce hendrerit nunc urna, vitae varius augue fringilla in. Nunc a ex eget lectus dictum mollis. Proin quis nisl consectetur metus bibendum ultricies eu non orci. Quisque nec efficitur quam. Suspendisse lorem nulla, sollicitudin ac sagittis id, eleifend eget eros. Interdum et malesuada fames ac ante ipsum primis in faucibus. Cras neque leo, consectetur nec sem quis, imperdiet viverra quam. Pellentesque ultricies felis a molestie egestas. Aliquam malesuada eget justo condimentum venenatis. Duis mattis ut nisi in suscipit. Phasellus scelerisque, arcu ut sollicitudin sagittis, ex ante posuere neque, et dapibus ex lectus quis libero. |

## Controlador


Como sempre, iremos começar com um simples controlador. De momento, o controlador deverá ser simples e ter apenas o seguinte:

```php
<?php

namespace Controller;

class Posts extends Controller
{

    function __construct()
    {
        parent::__construct('posts');
    }

    function index()
    {
        /* REQUERER POSTS */

        $this->view->setTitle('Posts');
        $this->view->render('posts/index');
    }

}
```

De momento, apenas declaramos o construtor e também a função index  que será aquela que vai  ser acedida ao acedermos a `URL/posts`.

## Modelo

Vamos então virar a nossa cara para os modelos. O modelo correspondente a este controlador deverá estar localizado em `app_core/models/posts.php` e deverá conter um código semelhante ao seguinte:

```php
<?php

namespace Model;

class Posts extends Model
{

    function __construct()
    {
        parent::__construct();
    }

    function getPosts()
    {
        return $this->db->select("SELECT * FROM posts");
    }
}
```

Onde utilizamos o construtor para criar uma ligação à base de dados e criamos a função `getPosts` que irá buscar todos as as colunas da tabela *posts* da base de dados a que está ligada a nossa aplicação.

Como pode ver, este é um modelo bastante simples cuja única função é buscar os ficheiros à base de dados. De momento não há mais nenhuma alteração.

## Voltando ao controlador...

Agora devemos voltar ao controlador e substituir o comentário que lá deixámos pelas seguintes duas linhas:

```php
$data = $this->model->getPosts();
$this->view->setData($data);
```

A função destas duas linhas é bastante simples: primeiro, declaramos uma variável que será igual ao retorno da função que criámos anteriormente para selecionar os artigos.

De seguida, "injetamos" o conteúdo desta variável na _view_ deste controlador utilizando a função `setData`.

## Que bela vista!

Agora só falta a parte que irá mostrar os itens na página: a _view_. Esta _view_ deverá ser declarada no diretório `app_core/views/posts/index.php`.

Relembro que a localização do ficheiro é definida quando utilizamos a função render . Então, o conteúdo que coloquei nesta página é o seguinte:

```php
<h1>Posts</h1>

<?php foreach ($this->_data as $post) : ?>
    <h2><?php echo $post['title']; ?></h2>
    <p><?php echo $post['content']; ?></p>
<?php endforeach; ?>
```

Como pode ver, aqui percorremos todos os itens do array `$_data` que faz parte da vista e, de seguida, imprimimos o título e o conteúdo de cada _post_.

## Ou seja,

Ou seja, é muito simples. Muitas coisas podem ser agora feitas neste modelo, pois a parte mais difícil já foi feita: o _core_, o _kernel_, o núcleo da aplicação.

**Tem alguma sugestão, ideia ou modificação? Sinta-se livre para contribuir para esta simples, pequena e _homemade framework_ no [GitHub](https://github.com/hacdias/InMVC) :)**