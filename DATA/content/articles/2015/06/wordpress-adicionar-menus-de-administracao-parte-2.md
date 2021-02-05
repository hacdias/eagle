---
description: Esta é a continuação do tutorial sobre como adicionar menus de administração a temas e plugins do WordPress.
publishDate: "2014-08-04T12:01:37.000Z"
tags:
- php
title: 'WordPress: adicionar menus de administração [Parte 2]'
updateDate: "2015-06-10T11:27:37.000Z"
---

O **WordPress** é, como vocês sabem, um dos mais populares CMS do mundo. Isto, se não o for. Um dos nossos primeiros artigos foi referente ao WordPress e ensinámos a [remover a hash dos links "Ler mais"](/articles/2014/07/remover-hash-dos-links-continuar-a-ler/).

Hoje vamos dar continuação ao tutorial sobre como [adicionar menus de administração](/articles/2014/08/wordpress-adicionar-menus-de-administracao/) a temas e plugins do WordPress.

<!--more-->

Na primeira parte, criámos duas funções e adicionámos uma opção ao menu de administração do WordPress sendo que ficámos com o seguinte código:

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

Hoje vamos dedicar-nos, principalmente, à segunda função (`themeslug_options()`) e à utilização de uma variável definida nas opções.

Resumindo, hoje vamos fazer o seguinte:

  * Criar uma opção das definições para adicionar o *link* do site para a sua página do Facebook.
  * Se o *link* estiver definido, vai aparecer um ícone no tema para ir para a página, caso não esteja, o ícone não aparecerá.

Pode parecer complicado, mas vai ver que é simples. Vamos começar por substituir o comentário que está dentro da função themeslug_options  pelo seguinte:

```php
<?php

//...

/*
 * Este loop if verifica se o utilizador tem permissões suficientes
 * para aceder a esta página. Se não tiver, a função termina aqui
 * utilizando a função wp_die() e será apresentada a mensagem que está
 * dentro de parêntesis ao utilizador.
 */
if (!current_user_can('manage_options')) {
    wp_die('Não tem permissões suficientes para aceder a esta página!');
}

?>

<!--  Aqui começa a nossa página, inserindo o título. -->
<div class="wrap">
    <h2>Opções</h2>

<?php

/*
 * Ao voltarmos ao PHP, declaramos uma variável chamada Facebook que vai
 * conter a opção que irá ser definida.
 *
 * Esta variável vai buscar a opção que está guardada. (Ver 'if' seguinte)
 */

$facebook = get_option("themeslug_facebook");

/*
 * Neste loop if, verificamos se o formulário abaixo foi submetido. Isto é feito
 * verificando se um "input" abaixo foi definido. Este estará oculto de forma a
 * que não possa ser alterado.
 *
 * Se estiver definida, a variável $facebook será "reescrita" colocando desta
 * vez o valor obtido através do método POST do formulário a seguir.
 */
if (isset($_POST["update_settings"])) {

    $facebook = esc_attr($_POST["facebook"]);

    /*
     * Aqui atualizamos a opção "themeslug_facebook" com o valor da variável
     * $facebook.
     */
    update_option("themeslug_facebook", $facebook);

    //Apresentação de uma mensagem de sucesso
    ?> <div id="message" class="updated below-h2"><p>Definições guardadas</p></div> <?php

}

/* Formulário que utilizaremos para submeter as definições.
 * Este formulário utiliza classes CSS próprias do WordPress,
 * como "top" e outras.  
 */

?>
<form method="POST" action="">
    <table class="form-table">
        <tr valign="top">
            <th scope="row">
                <label for="facebook">
                    Facebook:
                </label>
            </th>
            <td>
                <?php /* Este input será onde colocaremos o link. O seu valor será igual à variável $facebook
                    anteriormente definida. Assim, caso essa opção não tenha sido definida, será apresentado
                    um campo vazio.

                    Caso já tenha sido definida, o campo irá conter o valor atual tornando-se mais fácil de editar. */ ?>
                <input type="text" name="facebook" size="100%" value="<?php echo $facebook;?>"/>
            </td>
        </tr>
    </table>

    <?php /* Criação de um input oculto para informar o PHP se estamos a submeter o formulário ou não. */ ?>
    <input type="hidden" name="update_settings" value="Y" />

    <?php /* Botão de submissão do formulário. */ ?>
    <input type="submit" name="submit" value="Guardar alterações" class="button button-primary" />
</form>    <?php
```

O código acima está integralmente comentado. Se lerem os comentários de cima para baixo, não conhecendo o resto do código, é normal que sintam alguma confusão.

Porém, essa ordem é necessária de forma a que a variável `$facebook` fique definida e apareça no input  do formulário. Por isso, se tiverem alguma dúvida, não hesitem em perguntar.

Agora, quando acederem às definições do vosso tema (neste caso), irão ter uma página de opções.

Podem agora definir o *link* para a página do Facebook que desejam utilizar. Depois de guardarem as alterações ir-vos-à ser apresentada uma mensagem.

Para aplicar a variável no tema é muito simples. Como viram acima, utilizámos a função get_option  para obter a opção que foi gravada e será assim que a iremos obter e utilizar no tema.

O código acima verifica se a opção `themeslug_facebook` foi definida e se sim, cria um link em HTML que direcionará para a página que foi definida nas opções do tema.

É simples utilizar as opções nos temas do WordPress. Não se esqueçam, se não compreenderem qualquer coisa, podem sempre utilizar os comentários.

Espero que tenham gostado deste pequeno tutorial :)