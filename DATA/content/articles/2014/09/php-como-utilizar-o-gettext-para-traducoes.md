---
description: A linguagem PHP está repleta de extensões. Uma muito utilizada e útil é o Gettext que nos permite traduzir um site muito facilmente.
publishDate: "2014-09-21T11:07:03.000Z"
tags:
- php
title: PHP - Como utilizar o Gettext para traduções [Parte 1]
---

Como prometido no [último artigo](/articles/2014/09/php-wordpress-comandos-printf-sprintf/), hoje irá começar uma pequena saga de dois ou três artigos sobre como utilizar o Gettext no PHP.

Com o **gettext** podemos ter um site disponível em diversas linguagens de forma muito fácil e sem complicações. Nesta primeira parte, irá ser abordado como **configurar o gettext** e as suas **funções**.

<!--more-->

O gettext pode ser configurado no PHP através de uma extensão nativa ou através do uso de uma biblioteca separada.Nós iremos utilizar o segundo método por ser mais simples e fácil de configurar.

## Download e estruturação

Antes de mais nada, aviso que é **necessário** ter a extensão *mbstring* ativada nas definições do PHP. Em primeiro lugar, deve fazer o _download_ do **php-gettext** nesta [página](https://launchpad.net/php-gettext/).

Depois de efetuar o _download_, irá ter que descompactar o ficheiro transferido. Após a sua descompactação irá encontrar diversos ficheiros. Apenas iremos precisar dos seguintes:

  * `gettext.inc` → Aliases de funções para utilizar no sistema;
  * `gettext.php` → Funções do gettext;
  * `streams.php` → Classes e métodos que permitem ler ficheiros do gettext.

Agora crie uma pasta cujo conteúdo seja semelhante ao seguinte:

```bash
site-multi-lingua/
  | lib /
  |     | gettext.inc
  |     | gettext.php
  |      streams.php
  |
  | langs/
  |
  |
  | config.php
  | i18n.php
   index.php
```

Como pode visualizar, os ficheiros do gettext foram colocados dentro de uma pasta chamada `lib` e ainda foram adicionados três outros ficheiros:

  * `config.php` → faz o carregamento das configurações;
  * `i18n.php` → contém a inicialização do gettext;
  * `index.php`

Inicialmente tem que ser definida uma linguagem padrão. Nós iremos utilizar "pt_PT" ou seja, Português de Portugal.

Para isso, edite o seu ficheiro _config.php _e coloque o seguinte:

```php
<?php

define('LANG','pt_PT');
```

## Inicialização do Gettext

Agora que já existe uma linguagem padrão definida, a inicialização do gettext deve ser feita porém, primeiro tem que memorizar os seguintes conceitos:

  * `locale` → uma _string_ no formato `xx_XX` que indica a linguagem. "pt_BR" é português do Brasil, "en_US" corresponde a Inglês dos Estados Unidos. Pode ler mais sobre estes prefixos [aqui](https://pt.wikipedia.org/wiki/Internacionaliza%C3%A7%C3%A3o).

  * `textdomain` → quer dizer "domínio de texto" e é um local onde as traduções vão ser colocadas. No nosso caso, apenas iremos utilizar um _textdomain_.


Agora, deve colocar no ficheiro **i18n.php **o seguinte conteúdo:

```php
<?php

require_once('config.php');

$locale = LANG;
$textdomain = "site_multi_lingua";
$locales_dir = dirname(__FILE__) . '/lang';

if (isset($_GET['lang']) && !empty($_GET['lang']))
  $locale = $_GET['lang'];

putenv('LANGUAGE=' . $locale);
putenv('LANG=' . $locale);
putenv('LC_ALL=' . $locale);
putenv('LC_MESSAGES=' . $locale);

require_once('lib/gettext.inc');

_setlocale(LC_ALL, $locale);
_setlocale(LC_CTYPE, $locale);

_bindtextdomain($textdomain, $locales_dir);
_bind_textdomain_codeset($textdomain, 'UTF-8');
_textdomain($textdomain);

function _e($string) {
  echo __($string);
}
```

Poça! Tanta coisa! Mas para que serve tudo isto? Na **linha 3**, o ficheiro `config.php` é chamado pois é este que contém a constante da linguagem. Assim podemos já podemos utilizar essa constante neste ficheiro.

Nas linhas seguintes (**5 a 7**), a variável `$locale` é definida, tal como o domínio de texto e o local onde irão estar os ficheiros de tradução que, neste caso, será numa pasta chamada lang .

Mais à frente, nas **linhas 8 a 9**, verificamos se existe uma variável chamada `lang` a ser passada pelo URL. Se sim, reescrevemos o valor da variável `$locale` com essa linguagem.

Nas linhas seguintes, **12 a 5**, as variáveis de ambiente do sistema operativo são alteradas para o locale  que está a ser utilizado.

Na linha cuja posição é **17**, a biblioteca php-gettext é carregada sendo logo a seguir (**19 a 20**), o locale também carregado para o gettext.

Seguidamente, **nas linhas 22 a 24**, o textdomain é defido, tal como a codificação que será utilizada nesses ficheiros (UTF-8) e ainda dizemos onde vão estar as traduções.

De seguida é criada uma função chamada `_e()` que nos permite imprimir o resultado da função `__()`.

Agora, copiem e colem o seguinte no `index.php`:

```php
<?php
require_once('i18n.php');
?>

<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <title><?php echo __('Olá Mundo!'); ?></title>
  </head>
  <body>
    <h1><?php _e('Olá Mundo!'); ?></h1>
  </body>
</html>
```

De momento, ainda não existe nada de extraordinário a acontecer. Na próxima parte deste tutorial iremos falar em como criar as traduções!