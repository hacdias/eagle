---
description: Como fazer a listagem de uma tabela MySQL com interface web, botões de navegação e limitação do número de linhas por página.
publishDate: "2014-07-17T14:51:56.000Z"
tags:
- php
- html
- mysql
title: Listagem web de uma tabela MySQL
---

No [último artigo](/articles/2014/07/selecionar-um-intervalo-de-linhas-em-mysql/), falei sobre como selecionar um intervalo de linhas em MySQL e, no final, prometi escrever um tutorial que explicasse como criar um  pequeno sistema de navegação em PHP e HTML para listar as linhas contidas numa tabela de uma base de dados.

Aqui está ele. Abaixo encontram passo-a-passo como criar um pequeno sistema que liste todas as linhas de uma base de dados, separadas com páginas, cada uma com ```x``` linhas e botões de navegação.

<!--more-->

Para seguir este tutorial é necessário o Apache e o MySQL instalados no computador. Pode utilizar, por exemplo, o tudo-em-um [XAMPP](https://www.apachefriends.org/pt_br/index.html) que já conta com o Apache e MySQL e é muito fácil de instalar.

Para este tutorial estou a utilizar uma base de dados MySQL com uma tabela chamada `mytable` que conta com 145 linhas. Existem duas colunas: a do `id` e a do conteúdo a que chamei `content`.

Na primeira coluna, todas as linhas estão numeradas automaticamente e na segunda, todas as linhas têm uma pequena frase gerada [aqui](http://pt.lipsum.com/). O conteúdo desta coluna é igual para todas as linhas pois utilizei um pequeno script para criar todas as linhas automaticamente.


## 1. Ligação à Base de Dados


Neste tutorial assumo que os leitores sabem criar uma base de dados e/ou já têm uma criada. Caso tenham alguma dúvida não hesitem em perguntar através dos comentários.

Dentro da pasta onde estão colocados os documentos do servidor (o XAMPP usa normalmente a directoria "C:/xampp/htdocs") crie um ficheiro chamado **index.php**. Dentro deste copie e cole o seguinte código:

```php
<?php

/* Coloque abaixo os dados de acesso à base de dados.
 *
 * $host -> servidor onde está localizada a base de dados.
 *          se for o seu computador mantenha "localhost".
 *
 * $db   -> nome da base de dados.
 * $username -> o nome de utilizador para aceder à base de dados.
 * $password -> a senha correspondente ao utilizador anterior.
 */

$host = 'localhost';
$db = 'mydb';
$username = 'username';
$password = 'password';

//Utilizando a extensão PDO, criamos uma ligação à base de dados.
$db = new PDO('mysql:host=' . $host . ';dbname=' . $db, $username, $password);
```

Se não conhecerem bem a extensão PDO, recomendo-vos uma pequena leitura no [manual oficial do PHP](http://php.net/manual/pt_BR/book.pdo.php). Não se esqueça de alterar os dados acima para ligar à base de dados.

## 2. HTACESS e mais uns pormenores

Como vão ser listados `x` linhas por página, é provável que tenham que existir várias páginas. Estas irão ser do formato: `URL/NÚMERO_DA_PÁGINA`. Se estivermos a correr num servidor local (no nosso computador) será, por exemplo, `http://localhost/1`, `http://localhost/2`, etc.

Mas, quando escrevemos esse URL, o Apache assume que estamos a tentar aceder a pastas cujo nome é 1, 2... por aí fora. Para que isto não aconteça, vamos criar um ficheiro chamado `.htaccess` na mesma pasta onde está o `index.php`.

É crucial que o ficheiro tenha o nome mencionado. Neste copia e cola o seguinte código:

```apache
<IfModule mod_rewrite.c>
	RewriteEngine On

	RewriteCond %{REQUEST_FILENAME} !-d
	RewriteCond %{REQUEST_FILENAME} !-f
	RewriteRule ^(.*) index.php?url=$1 [QSA,L]
</IfModule>
```

Normalmente o `mod_rewrite` já vem ativado por padrão. Caso vocês tentem aceder a alguma página e dê algum erro, expliquem o que acontece nos comentários para eu ou outro leitor tentarmos ajudar.

Não vou explicar de forma profunda o código acima, mas este faz com que todo o que escrevemos depois da barra do URL seja aberto na **index.php** através de um método GET. Podes ler mais sobre os métodos GET [aqui](http://www.w3schools.com/tags/ref_httpmethods.asp).

Agora precisamos que o nosso ficheiro em PHP consiga saber qual a página em que estamos de forma a mais tarde mostrar apenas os itens referentes aquela página. Para isso copia e cola o código seguinte a seguir da ligação à base de dados:

```php
/*
 * Utilizando o operador ternário, definimos
 * a variável $url.
 */

$url = isset($_GET['url']) ? $_GET['url'] : null;

//Remove-se alguma barra do final do $url caso exista.
$url = rtrim($url, '/');

// Cria-se uma array dividindo o $url em partes.
$url = explode('/', $url);

/*
 * Se o primeiro valor da array estiver vazio,
 * define este como 1 pois não podemos ter uma página sem identificação.
 */

if (empty($url[0])) {
      $url[0] = 1;
}

/*
 * Definimos a variável $n com o primeiro valor que vem depois do URL, o número da página.
 * $n -> Número da Página
 */

$n = $url[0];
```

A explicação do código acima está feita nos comentários. Podes saber mais acerca do **operador ternário** [aqui](/articles/2014/07/operador-ternario/). Agora que já podemos saber o número da página e temos uma ligação à base de dados pronta, vamos avançar para a página e a listagem em si.


## 3. Um pouco de HTML... e PHP


Abaixo da tag ?> , copiem e colem o seguinte trecho de código HTML.

```html
<html>
	<head>
		<title>Listagem de uma tabela MySQL</title>
	</head>

	<body>
		<h1>Listagem de uma tabela MySQL</h1>
		<!-- Aqui vamos criar a listagem -->

	</body>
</html>
```

Penso que tudo o que está acima seja conhecido de todos vós por isso, vamos continuar.  O restante código será todo escrito no local do comentário acima. Faremos a listagem das linhas e os botões de navegação aí.

Agora, no local do comentário copia o seguinte:

```php
<?php

/*
 * Criação de uma consulta à Base de Dados para
 * sabermos o número total de linhas existentes na tabela
 */
$consulta = $db->query("SELECT count(*) FROM mytable");
$numeroDeLinhas = $consulta->fetchColumn();

/*
 * Sabendo o número de linhas é possível determinar o
 * número máximo de páginas. Para isso dividimos o número
 * total de linhas pelo número de itens a apresentar em
 * cada página. ($numeroDeLinhas / $itensPorPag).
 *
 * Depois arredondamos para cima esse número utilizando
 * a função ceil().
 *
 * O arredondamento para cima é feito porque, por exemplo,
 * se a divisão resultasse em 4.2, seriam necessárias 5 páginas
 * para mostrar todo o conteúdo e não 4. Utilizando um arredon-
 * damento tradicional, o valor ficaria em 4 não mostrando os
 * restantes itens.
 */

$itensPorPag = 15;
$maximoDePaginas = ceil($numeroDeLinhas / $itensPorPag);

if ($n > $maximoDePaginas || $n < 1) {

	/*
	 * Se o número da página for superior ao número máximo de páginas
	 * ou for menor que 1, a página mostra a mensagem que está entre aspas.
	 */

	echo 'Página não existe.';

} else {

	/*
	 * Por outro lado, se o número da página for coerente, devemos definir
	 * o OFFSET para utilizar na consulta à BD.
	 *
	 * Para definir o offset/deslocamento utilizamos uma pequena função algébrica
	 * sendo a variável $n o número da página, substraímos a esta 1 e multiplicamo-la
	 * pelo número de itens por página.
	 *
	 * Exemplos:
	 *
	 *	Sendo $n = 1 e $itensPorPag = 15, $offset = (1-1)*15 = 0*15 = 0.
	 *	O deslocamento seria 0 e o limite 15 logo seriam apresentados os itens 1 a 15.
	 *
	 *	Sendo $n = 2 e $itensPorPag = 15, $offset = (2-1)*15 = 1*15 = 15.
	 *	O deslocamento seria 0 e o limite 15 logo seriam apresentados os itens 16 a 30.
	 */

	$offset = ($n - 1) * $itensPorPag;

	/*
	 * Aqui é feita a consulta à Base de Dados que seleciona o intervalo de linhas
	 * referentes à página atual.
	 */

	$query = "SELECT * FROM mytable LIMIT ". $itensPorPag . " OFFSET " . $offset;
	$items = $db->query($query);

	foreach($items as $item)	{	?>

		<!--
			O apresentado abaixo é o que vai ser mostrado para cada linha que foi
			seleciona com a query anterior.

			O formato para chamar o conteúdo de uma coluna de uma linha é:

			$item['NOME_DA_COLUNA'];
		-->

		<h2>Id: <?=$item['id'];?></h2>
		<p><?=$item['content'];?></p>

	<?php }

	//Botões de Navegação

	if ($n > 1) {

		/*
		 * Se o  número da página for maior que 1, mostra o botão para voltar
		 * à página anterior.
		 */

		$pagAnterior = $n - 1; ?>

		<button><a href="/<?=$pagAnterior;?>">Anterior</a></button>

	<?php }

	if ($n < $maximoDePaginas) {

		/*
		 * Se o número da página for inferior ao número total de páginas,
		 * mostra o botão para ir para a página seguinte-
		 */

		$pagSeguinte = $n + 1; ?>

		<button><a href="/<?=$pagSeguinte;?>">Seguinte</a></button>

	<?php }

}

?>
```

O código está todo comentado e explicado. Reforço novamente que, caso tenham  alguma dúvida, não hesitem em perguntar. Se quiserem saber mais sobre a função ceil , podem aceder a [esta página](http://php.net/manual/en/function.ceil.php).


## 4. Já está

Tal como o sub cabeçalho indica, já está! Basta abrirem a página através do browser para verem como ficou. É evidente que essa página não está muito bonita mas vocês logo tratam do CSS.

Visto que o PDO é um pouco abstrato às bases de dados, este tutorial funciona também para **SQLite** e outros tipos de Bases de Dados alterando apenas o comando de conexão.

Espero que tenham gostado :)