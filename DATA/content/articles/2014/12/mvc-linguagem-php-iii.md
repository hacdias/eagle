---
description: No seguimento da criação de uma framework PHP que utilize o modelo MVC, hoje explicamos como se criam os controladores.
publishDate: "2014-12-20T09:00:23.000Z"
tags:
- MVC
- PHP
- Tutorial
title: MVC na linguagem PHP - III
---

A pedido de um leitor, decidimos escrever um artigo em que exemplificamos este maravilhoso (ou não) modelo MVC utilizando a linguagem de programação PHP.

A [primeira](/articles/2014/12/mvc-linguagem-php/) e [segunda](/articles/2014/12/mvc-linguagem-php-ii/)  partes já foram publicadas e hoje é a vez da terceira onde tudo começa a tornar-se mais facilmente compreendido.

<!--more-->

Em primeiro lugar, quero referir algumas modificações que efetuei em alguns dos ficheiros que já trabalhámos.

Adicionei a constante `SITE_TITLE` ao ficheiro `config.php`, ficando a primeira secção deste ficheiro da seguinte forma:

```php
<?php //...
//1. Base Constants
define('ROOT', dirname(__FILE__) . '/');
define('URL', 'http://localhost/mvc/');
define('SITE_TITLE', 'Simple MVC Structure Model');
//...
```

Na classe `View` foi adicionada uma função que permitirá ao utilizador definir o título (aquele que aparece na parte superior do _browser_).

```php
public function setTitle($title)
{
    $this->_pageInfo['title'] = $title . ' | ' . SITE_TITLE;
}
```

## ```public_html```

Hoje vamos continuar a nossa jornada começando na pasta pública, ou seja, na pasta `public_html`. Vamos começar com o nosso maravilhoso `.htaccess`.

```apache
ErrorDocument 404 /404
ErrorDocument 500 /500

<IfModule mod_rewrite.c>

    RewriteEngine On

    RewriteCond %{REQUEST_FILENAME} !-d
    RewriteCond %{REQUEST_FILENAME} !-f
    RewriteCond %{REQUEST_FILENAME} !-l

    RewriteRule ^(.+)$ index.php?url=$1 [QSA,L]

</IfModule>
```

Como pode ver, todos os pedidos serão direcionados para o ficheiro `index.php` com o URL completo na forma de parâmetro GET.

Falando em `index.php`, e que tal lhe darmos uma espreitadela? Este vai ser o ficheiro que vai iniciar toda a sequência de acontecimentos. Ora veja:

```php
<?php

/**
 * Main file.
 *
 * @package MVC PHP Bootstrap
 */
require '../app_core/config.php';

function autoLoad($className)
{
    require DIR_LIBS . $className . '.php';
}

spl_autoload_register('autoLoad');

$bootstrap = new Bootstrap();
$bootstrap->init();
```

Neste ficheiro, como pode ver, o ficheiro das configurações é chamado e, de seguida, é criado uma função de auto carregamento.

O que faz esta função? Simplesmente tenta carregar uma classe indefinida. Como pode ver, eu não importo a classe `Bootstrap`, mas logo de seguida uso-a.

Consigo utilizá-la porque a a função de *auto load* chama esta classe automaticamente. Mas atenção! Este código apenas nos permite carregar automaticamente classes contidas na pasta `libs`.

> Se neste momento experimentar correr o código nesta pasta irá receber diversos erros porque ainda não foram criados nenhuns controladores. Nem mesmo o dos erros!

## *Header and Footer*

Antes de continuarmos para a criação dos controladores, devemos criar os ficheiros do cabeçalho e do rodapé que são chamados quando executamos a função _render_.

Estes dois ficheiros deverão estar localizados na diretoria `app_core/views/*.php`, onde `*` corresponde a _header_ e a _footer_.

A minha proposta para o ficheiro do cabeçalho é a seguinte:

```html
<!doctype html>
<html lang='en-EN'>

<head>
    <title><?php echo (isset($this->_pageInfo['title'])) ? $this->_pageInfo['title'] : SITE_TITLE; ?></title>

    <meta charset='utf-8'>
    <meta name='viewport' content='width=device-width, initial-scale=1'>

    <link rel='stylesheet' href='<?php echo URL; ?>css/template.css' type='text/css' media='all'/>
    <link rel='stylesheet' href='<?php echo URL; ?>css/normalize.css' type='text/css' media='all'/>
</head>

<body>
<div id='header'>
    <strong>MVCPHPB</strong>
    | <a href="<?php echo URL; ?>">Home</a>
    | <a href="<?php echo URL; ?>page">Page</a>
    | <a href="<?php echo URL; ?>posts">Posts</a>
</div>

<div id="wrap">
```

Onde já temos o título da página que será igual ao título que é definido utilizando a função que referi no início do artigo, ou apenas o nome do site.

Deixei também uma pequena introdução à estrutura HTML do site e uma pista àquilo que iremos criar: iremos criar uma página estática (_Page_) e uma página dinâmica com alguns itens de uma base de dados (_Posts_).

Estou também a utilizar o `normalize.css` e uma folha de estilos própria com alguns ajustes de forma a distinguir melhor cada uma das partes do site. Aqui está o `template.css` (deve ser colocado na pasta `css`):

```css
#header,
#wrap,
#footer {
  padding: 20px;
}

#header {
  background: rgba(0,0,0,0.12);
}

#footer {
  background: rgba(0,0,0,0.5);
  }
```

Vamos agora ver o rodapé. O rodapé está localizado no ficheiro `app_core/views/footer.php` e o conteúdo deve ser algo semelhante ao seguinte:

```html
</div>

<footer id="footer">
    <strong><strong>MVCPHPB</strong></strong> Footer
</footer>

</body>
</html>
```

## Controladores

Depois de termos "montado" a estrutura, devemos passar à criação das páginas. Em primeiro lugar, vamos criar o controlador do erro.

```php
<?php

namespace Controller;

class Error extends Controller
{

    function __construct()
    {
        parent::__construct('error');
    }

    function index($error = '404')
    {
        $data = array();

        switch ($error) {
            case '404':
                $data['title'] = 'Error 404';
                $data['msg'] = "Not found. There is nothing here.";
                break;
            case '500':
            default:
                $data['title'] = 'Error 500';
                $data['msg'] = "Internal Server Error. Probably we did something wrong.";
                break;
        }

        $this->view->setTitle($data['title']);
        $this->view->setData($data);
        $this->view->render('error/index');
    }

}
```

Os controladores devem pertencer à *namespace* Controller e devem ter, obrigatoriamente, o seu construtor presente no formato mostrado acima.

Este é um controlador que só terá o método _index _relacionado com as páginas pois não irá ser utilizado mais nenhum. Acima pode visualizar que este método irá receber um argumento que, por padrão, é '404'.

De seguida, algumas informações são atribuídas dependendo do erro em questão. Depois o título da página é definido e, mais tarde, os dados são enviados para a _view_.

Falando em _view_, esta é renderizada logo de seguida. A _view _que colocámos é `error/index`, então o ficheiro PHP deverá encontrar-se em `app_core/views/error/index.php`.

> Se experimentar correr o código neste momento, deverá receber um erro por não encontrar a _View _da página de erro.

Vamos então visualizar a _view_ que corresponde à página de erro:

```php
<h1> <?php echo $this->_data['title'] ?></h1>

<p><?php echo $this->_data['msg']; ?></p>
```

É simples. Serve apenas para mostrar que ocorreu um erro. Mas não podemos continuar assim pois ao abrirmos a página inicial estamos apenas a ver um erro.

Vamos então criar o controlador da página inicial, que deve conter o seguinte código:

```php
<?php

namespace Controller;

class Index extends Controller
{

    function __construct()
    {
        parent::__construct('index');
    }

    function index()
    {
        $this->view->setTitle('Home');
        $this->view->render('index/index');
    }

}
```


E o ficheiro da sua _view _deverá ter qualquer coisa que deseje. Eu, por exemplo, coloquei:

```html
<h1>MVC PHP Bootstrap (MVCPHPB)</h1>

<p>This is the main page of this website that is a simple mvc structure model example.</p>

<p>Esta é a página principal deste site que é um exemplo de modelo de estrutura MVC simples.</p>
```

Coloque o que queira. De momento, se aceder à página inicial deverá visualizar a página inicial propriamente dita e se tentar, eventualmente, abrir qualquer outra possível página, irá ver a página de erro 404.

**No próximo tutorial iremos ver a criação de uma página dinâmica com um modelo**. Pode, entretanto, experimentar criar mais páginas ou até mesmo criar mais métodos para sub-páginas.