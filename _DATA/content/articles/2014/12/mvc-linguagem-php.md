---
description: O modelo MVC é algo muito utilizado atualmente. Nesta primeira parte iremos explicar a estrutura base de uma aplicação MVC e criar o ficheiro principal
publishDate: "2014-12-18T22:55:34.000Z"
tags:
- php
title: MVC na linguagem PHP
---

No último artigo publicado, foi falado acerca do [**Modelo MVC**](/articles/2014/11/mvc-uma-breve-explicacao/) e sobre aquilo em que este consiste. O Modelo _Model-View-Controller_ é amplamente utilizado nos dias de hoje.

A pedido de um leitor, decidimos escrever um artigo em que exemplificamos este maravilhoso (ou não) modelo MVC utilizando a linguagem de programação PHP.

<!--more-->

Gostaria de dizer que esta exemplificação terá como base uma "framework base" MVC construída por mim através de vários tutoriais e ideias que encontrei _online_.

Para a aplicação começar a ser construída, é necessário efetuar a estruturação da mesma. Esta estruturação passa pela disposição dos ficheiros pelas pastas. Irá ser utilizada a seguinte estrutura:

```bash
| app_core (Application Core)
     | controllers
     | libs
     | models
     | views
     | config.php

| public_html (Public HTML)
     | css
     | imgs
     | js
     | .htaccess
     | index.php
```

Visualizando a estruturação anterior, é possível verificar que existem duas pastas principais: a pasta `app_core` e a pasta `public_html`. Cada uma das duas pastas tem a sua função específica. Vejamos então qual a finalidade de cada uma das pastas.


## *Application Core*

O núcleo (ou motor) da aplicação está contido na pasta `app_core` estando no mesmo nível que a pasta que irá estar pública através do _browser_, logo os visitantes não terão acesso a nenhum conteúdo desta pasta. Dentro desta pasta podemos verificar a existência de quatro outras e também de um ficheiro.

  * `controllers` → pasta para colocar os controladores da aplicação que são, de forma generalizada, aqueles ficheiros que comandam a aplicação: o seu cérebro;

  * `libs` → nesta pasta estão contidas as classes base para todos os outros ficheiros da aplicação como os modelos, controladores, vistas, base de dados, etc.

  * `models` → aqui são colocados os modelos da aplicação, ou seja, aqueles ficheiros que estão encarregados da manipulação de dados;

  * `views` → esta pasta irá conter todos os ficheiros que têm como base HTML. Mais nenhuma pasta da aplicação deverá conter ficheiros com HTML;

  * `config.php` → o ficheiro de configuração principal. Aqui são definidas as diversas constantes que irão ser necessárias na execução da aplicação.

## *Public HTML*

A pasta `public_html` será aquela que para a qual o servidor web, como por exemplo o apache, estará a apontar. Esta irá conter todos os ficheiros que estarão disponíveis ao utilizador.

  * `css` → todos os ficheiros de estilo serão incluídos nesta pasta;
  * `imgs` → qualquer imagem que seja utilizada na aplicação poderá ser colocada aqui;
  * `js` → todos os scripts escritos na linguagem _javascript_ deverão aqui ser colocados;
  * `.htacces`s → o ficheiro que dispensa apresentações;
  * `index.php` → o ficheiro onde tudo começa.

## config.php

Iremos começar pelo ficheiro mais simples: o ficheiro `config.php` que está contido base do núcleo da aplicação (`app_core/config.php`). Este ficheiro, como já referido, irá conter as constantes principais. Ora veja:

```php
<?php

/*
* Ficheiro de Configuração
*
* Este ficheiro contém a configuração base deste website/aplicação:
*  1. Constantes Base
*  2. Constantes de Diretórios
*  3. Configuração da Base de Dados
*  4. Definições de erros
*
* Coloque sempre uma barra (/) depois de todos os caminhos.
*/

//1. Constantes Base
define('ROOT', dirname(__FILE__) . '/');
define('URL', 'http://localhost/mvc/');

//2. Constantes de Diretórios
define('DIR_LIBS', ROOT  . 'libs/');
define('DIR_MODELS', ROOT  . 'models/');
define('DIR_VIEWS', ROOT  . 'views/');
define('DIR_CONTROLLERS', ROOT . 'controllers/');
define('DIR_PUBLIC', '../public_html/');

//3. Configuração da Base de Dados
define('DB_TYPE', 'mysql');
define('DB_HOST', 'localhost');
define('DB_NAME', 'dbname');
define('DB_USER', 'dbuser');
define('DB_PASS', 'dbpass');

/*
* 4. Definições de Erros
*
* Defina error_reporting:
*  para -1 de forma a mostrar todos os erros que ocorrem;
*  para  0 de forma a ocultar todos os erros gerados.
*/
error_reporting(-1);
ini_set( 'display_errors','-1');
```

Crie um ficheiro com o código acima e guarde-o na pasta mencionada. Não se esqueça de alterar a constante URL, de forma a corresponder com o URL que vai utilizar para criar esta pequena aplicação. Altere também os dados da Configuração da Base de Dados.


### Dica:


Como pode ter reparado acima, eu estou a utilizar o link `localhost/mvc`  que redirecionará para a pasta pública da aplicação, sendo que a base dos documentos do meu servidor é `C:\Web\Server\Apache24\htdocs`  e a aplicação está localizada em `D:\Development\mvc`.

Como fiz isto? Criei um [*link simbólico*](http://en.wikipedia.org/wiki/Symbolic_link) de forma a que `C:\Web\Server\Apache24\htdocs` corresponda a `D:\Development\mvc`. Para criar um link simbólico no Windows basta correr o seguinte comando na linha de comandos em modo de administrador:

```bash
MKLINK /D <novo-link> <local-dos-ficheiros>
```

Onde `<novo-link>` corresponde ao local a que irá corresponder o `<local-dos-ficheiros>`. No meu caso, tive que executar o seguinte comando:

```bash
MKLINK /D C:\WebServer\Apache24\htdocs\mvc D:\Development\mvc\public_html
```

Isto pode ser, obviamente, feito também em _linux_ e _OS X_ porém não sei como, mas caso tenha uma distribuição Linux ou OS X, recomendo a leitura desta [página](http://apple.stackexchange.com/questions/115646/how-can-i-create-a-symbolic-link-in-terminal).

## Saindo da dica...

Como pode ver pela estrutura da aplicação, ainda há muito a fazer. Em breve lançarei a segunda parte desta mini-série de artigos. Esperemos que tenham gostado deste e que gostem dos próximos artigos.