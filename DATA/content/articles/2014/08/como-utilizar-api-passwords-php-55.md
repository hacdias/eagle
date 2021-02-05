---
description: A API de passwords introduzida na versão 5.5 do PHP é excelente. Trouxe quatro novas maravilhosas funções. Resumimos a forma como cada uma funciona.
publishDate: "2014-08-20T22:16:15.000Z"
tags:
- php
- security
title: Como utilizar a API de Passwords do PHP 5.5+
---

Como muitas das novidades introduzidas na [versão 5.5](http://php.net/manual/en/migration55.changes.php) do PHP, a API de passwords não passou despercebida aos desenvolvedores. Hoje vamos falar da criação de hash de passwords utilizando esta API.

<!--more-->

Antes de começarmos a analisar como se criam as passwords, vamos ver que novas funções trouxe [esta API](http://php.net/manual/en/book.password.php) para o PHP:

  * `password_get_info`
  * `password_hash`
  * `password_needs_rehash`
  * `password_verify`

Cada uma tem uma função diferente porém estão todas relacionadas. Assim, vamos falar de cada uma destas funções. Vamos começar!

## password_hash

Esta função pode ser considerada a principal do conjunto pois é com ela que criamos as *hash* e deve ser utilizada da seguinte forma:

```php
<?php

$password = 'a_minha_password';

/*
 * Da seguinte forma podemos gerar a hash de uma password
 * utilizando o algoritmo que está definido em DEFAULT.
 *
 * Na versão 5.5.0 do PHP este algoritmo correponde ao BCRYPT
 *
 * A hash vai ter o comprimento de, no mínimo, 60 caracteres.
 * Este comprimento pode alterar em novas versões do PHP.
 */
password_hash($password, PASSWORD_DEFAULT);
```


Podemos ainda definir alguns parâmetros opcionais nomeadamente o ```cost``` ("custo") e o ```salt``` ("sal") da seguinte forma:

```php
<?php

$password = 'a_minha_password';

/*
 * Da seguinte forma podemos gerar a hash de uma password
 * utilizando o algoritmo BCRYPT.
 *
 * Podemos definir duas opções no terceiro parâmetro:
 *  COST  =>  Por padrão, será criado um novo por cada nova
 * hash criada porém pode ser definido.
 *
 *  SALT  =>  Por padrão, o valor 10 irá ser utilizado porém
 * pode ser alterado à semelhança do anterior.
 */

$options = array(
    'cost'  => 13,
    'salt'  => mcrypt_create_iv(25, MCRYPT_DEV_URANDOM)
  );

password_hash($password, PASSWORD_BCRYPT, $options);
```

Acrescento ainda que a função retornará ```false``` em caso de erro.


## password_verify

Depois de criarmos a *hash* vamos precisar de, evidentemente, confirmá-la. Para confirmar se uma password corresponde a uma hash basta utilizarmos esta função da seguinte forma:

```php
<?php

/*
 * Para utilizar esta função basta colocar a password
 * a verificar no primeiro parâmetro e a hash no segundo.
 *
 * Não precisa de se preocupar se colocou algum "salt" ou
 * "cost" personalizados pois esta informação está embutida
 * na hash.
 *
 * A função irá returnar os valores "true" ou "false".
 */

$password = 'teste';
$hash = '$2y$10$M.3t0/gmB12IrSETmINf7uy9XhruDrmB8vjaktfd5vC8AfVPH695.';

password_verify($password, $hash);
```

## password_get_info

Esta informação permite-nos obter a informação acerca de uma hash. A informação obtida corresponde ao algoritmo, o seu nome e as opções dadas quando criámos a hash. Exemplo:

```php
<?php

//Criação de uma hash de exemplo

$options = array(
    'cost'  => 13,
    'salt'  => mcrypt_create_iv(25, MCRYPT_DEV_URANDOM)
  );

$hash = password_hash('a_minha_password', PASSWORD_BCRYPT, $options);

/*
 * Utilizando a função password_get_info, vamos obter as diversas opções
 * enviadas quando criámos a hash.
 */

$info = password_get_info($hash);

/*
 * Assim, var_dump($info) irá retornar o seguinte:
 *
 * array(3) {
 * 	["algo"]=> int(1)
 *	["algoName"]=> string(6) "bcrypt"
 *	["options"]=> array(1)
 *		{ ["cost"]=> int(13) }
 *	}
 */
```

## password_needs_rehash

Esta função permite-nos confirmar se uma hash já criada corresponde a uma array de opções e a um algoritmos. Podem observar mais no seguinte exemplo:

```php
<?php

//Criação de uma hash de exemplo

$options = array(
    'cost'  => 13,
    'salt'  => mcrypt_create_iv(25, MCRYPT_DEV_URANDOM)
  );

$hash = password_hash('a_minha_password', PASSWORD_BCRYPT, $options);

/*
 * A função seguinte irá retornar true porque todas as opções corresponde
 * às que foram dadas anteriormente.
 */

password_needs_rehash($hash, 0, $options);
```

Se tiver qualquer dúvida relacionada ao segundo parâmetro da função, sugiro-lhe a leitura [desta página](http://php.net/manual/pt_BR/password.constants.php) e também que verifique os valores obtidos com ```password_get_info```.

Espero que dê bom uso a esta função :)