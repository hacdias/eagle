---
description: Os DOCBlocks do PHP são extremamente úteis para os desenvolvedores que desejam partilhar o código. Venha conhecer as tags mais utilizadas.
publishDate: "2014-08-11T20:26:21.000Z"
tags:
- php
title: PHP - (Algumas) tags dos DOCBlocks
---

Para quem desenvolve aplicações em PHP com outras pessoas ou mesmo sozinho, por vezes precisa de colocar **comentários** em vários locais para identificar os diversos processos que vão ocorrendo.

Os **DocBlocks** são (quase) fundamentais na vida dos programadores e estão disponíveis em várias linguagens. Vamos analisar as **tags** que existem nos de PHP e como os utilizar.

<!--more-->

Os DocBlocks em PHP são parte do PHPDoc, uma adaptação do sistema **javadoc** para a linguagem de programação PHP.

Os DocBlocks, ao contrário dos comentários tradicionais, começam sempre por `/**` ao invés de `/*` ou `//`.

Estes blocos documentam o código a que precedem. Abaixo podem ver um simples exemplo:

```php
<?php

/**
 * Função lorem faz x,y,z
 *
 * @param    string $sth    Descrição do param
 * @return    string        Descrição do retorno
 */
function lorem($sth = '') {

    if (!is_string($sth)) {
        return 'Fail';
    }

    return $sth;
}
```

Normalmente, a estrutura destes blocos de documentação é:

  * Descrição curta
  * Descrição longa
  * Tags

Hoje vamos analisar o último ponto: as **tags**. Existem imensas e vamos apenas ver as principais e mais utilizadas.

```php
<?php

/**
 * @author        Nome Do Autor <email@do.autor>    -> Autor do ficheiro
 * @copyright     Nome Data                         -> Info da Copyright
 * @param         tipo [$nome-da-var] descrição     -> Info acerca de um parâmetro
 * @return        tipo descrição                    -> Info acerca do retorno de uma função, p.e.
 * @since         Versão                            -> Disponível desde a versão xxxx
 * @todo          Descrição do afazer               -> Tarefas para fazer
 * @package       Nome do pacote                    -> Nome do pacote onde o ficheiro está inserido
 * @subpackage    Nome do sub-pacote                -> Nome do sub-pacote
 * @deprecated    Versão                            -> Definição de um método obsoleto a partir da versão xxxx
 * @version       Versão                            -> Utilizado para definir a versão de um ficheiro/método
 */
 ```

O código acima pode ser visivelmente dividido em três colunas. A primeira são as **tags**, a segunda a forma de **implementação** e a terceira, que começa com setas, são indicações a descrever para que servem as tags.

Estas informações são apenas para vos informar e não devem ser utilizadas em quaisquer contexto dentro de um ficheiro PHP.

Existem mais tags que podem ser inseridas nos comentários DOCBlocks porém as que se encontram acima são as mais importantes e essenciais. Se quiseres ler mais sobre isto, podes aceder a [esta página.](http://manual.phpdoc.org/HTMLSmartyConverter/PHP/phpDocumentor/tutorial_tags.pkg.html)

Espero que este artigo  vos tenha sido útil.