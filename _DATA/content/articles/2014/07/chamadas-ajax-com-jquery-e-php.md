---
description: Efetuar pedidos POST e GET utilizando a função ajax do jQuery e responder utilizando a linguagem de servidor PHP.
publishDate: "2014-07-20T10:41:11.000Z"
tags:
- php
- javascript
title: Chamadas Ajax com jQuery e PHP
---

A biblioteca [jQuery](http://jquery.com/) é das mais conhecidas e utilizadas em JavaScript. Com esta biblioteca podemos proceder a chamadas **ajax** muito facilmente através de poucas linhas de código.

Vamos aprender a efetuar pedidos POST e GET com Ajax, obtendo os dados do lado do servidor utilizando PHP e responder a essa mesma chamada.

<!--more-->

Iremos analisar um exemplo prático: um  formulário de registo ou inscrição  utilizando as tags ```<form>``` do HTML, Ajax para enviar os pedidos para o servidor e  PHP processar os dados e enviar a resposta novamente para o lado do cliente onde será mostrada uma mensagem de sucesso ou erro conforme o sucedido.

Como vamos utilizar o método Ajax do jQuery, temos que importar, em primeiro lugar, a biblioteca jQuery. Na secção <head>  do vosso ficheiro onde vai estar o registo - vou chamar-lhe ```index.html``` - devem colocar o seguinte código:

```html
<script src="//ajax.googleapis.com/ajax/libs/jquery/2.1.1/jquery.min.js"></script>
```

Esse pequeno trecho importa a versão 2.1.1 do jQuery e está armazenada nos servidores da Google e o seu uso é livre e qualquer um pode utilizar. Para criar o formulário de registo, vamos utilizar as tags ```<form>``` do HTML, criando assim, algo semelhante ao seguinte:

```html
<h1>Formulário de Registo</h1>

<form id="registo" method="post" action="javascript:enviarRegisto();">

	<input type="text" id="nome" name="nome" placeholder="Nome"><br>
	<input type="password" id="password" name="password" placeholder="Password"><br>
	<input type="email" id="email" name="email" placeholder="Email"><br>
	<input type="submit" value="Submeter">

</form>
```

O título não é necessário para o  pleno funcionamento do formulário. Como podem ver, todos os <inputs>  estão identificados com um id. No final encontra-se um botão para submeter o formulário. Este botão direciona para a ação do formulário que, neste caso, é javascript:enviarRegisto(); .

O método (method) não é necessário visto que este vai ser definido diretamente através do pedido em Ajax. Agora vamos à parte do servidor - registar o utilizador. Vamos criar um ficheiro com o nome ```processar.php``` o mesmo local onde temos o nosso ```index.html```. Neste ficheiro teremos o código seguinte ou semelhante:

```php
<?php

/*
 * Receber os dados do formulário através
 * de informações enviadas pelo ajax com
 * o método POST.
 */
$nome = $_POST['nome'];
$password = $_POST['password'];
$email = $_POST['email'];

/*
 * Criação de uma variável que mais tarde irá
 * guardar o resultado da operação: se foi concluída
 * com sucesso ou não.
 */
$resultado =  array();

/*
 * Ligação à base de dados utilizando PDO. Eu, por exemplo,
 * utilizei SQLite mas  pode ser utilizado qualquer outro tipo
 * de bases de dado.
 */
$db = new PDO('sqlite:db.sqlite');


/*
 * Query/Chamada para inserir os dados que obtemos via POST
 * na base de dados.
 */
$query = "INSERT INTO utilizadores VALUES ('" . $nome . "', '" . $password . "', '" . $email . "');";

if($db->query($query)) {

	/*
	 * Se a chamada for concluída com  sucesso,
	 * será atribuído o valor "true" ao elemento
	 * status da array $resultado.
	 */

	$resultado['status'] = true;

} else {

	//Caso contrário será falso.

	$resultado['status'] = false;
}


/*
 * Informa que o arquivo vai ser do tipo Json.
 * Assim, o Ajax vai conseguir receber a resposta
 * corretamente.
 */

header('Content-type: application/json');

/*
 * Envio da array $resultado novamente para o lado do cliente
 * em  formato json.
 */
echo json_encode($resultado);

?>
```


Podes ler mais sobre o formato **json** [aqui](http://json.org/). Antes de continuarmos, vamos estabelecer as diferenças entre os métodos POST e GET. Talvez a maior diferença entre estes dois métodos seja a visibilidade. O método **GET** leva a informação "agarrada" ao URL e qualquer pessoa pode ver. Os URLs com informações anexadas são do seguinte formato:

```txt
http://example.com?infoNome=info&infoNome2=info2
```

Por outro lado, o método **POST **leva a informação encapsulada no corpo do pedido e não pode visualizada. Uma desvantagem deste método é ser mais lento visto que é encapsulado ao contrário do GET, que é mais simples. Como os pedidos GET são feitos através do URL, existe uma limitação no comprimento da mensagem enviada sendo que não pode ter mais de 255 caracteres dependendo do browser.

O **tipo de dados** é outra grande diferença: enquanto que o método GET só pode enviar texto, o método POST pode enviar qualquer tipo de informação. Podes saber mais sobre estes dois métodos [aqui](http://www.w3schools.com/tags/ref_httpmethods.asp). Agora que já temos o formulário de registo e o servidor preparado, só falta fazer a ligação entre o cliente e o servidor utilizando Ajax.

Em primeiro lugar, criamos um novo ficheiro chamado, por exemplo, ```script.js``` e chamomo-lo logo a seguir ao jQuery no nosso ```index.html``` com um código semelhante ao seguinte:Agora, dentro do ficheiro de javascript, vamos ter que criar a função que anteriormente colocámos na ação do formulário, a função enviarRegisto().

```javascript
function enviarRegisto() {

    /*
     * Obtenção dos dados do formulário e colocação dos mesmos
     * no formato nomeDaInfo=Info para enviar por POST.
     *
     * Utiliza-se a função val() para obter os valores
     * dos inputs com os id's em questão.
     */


    /*
     * Criação da variável data que vai conter toda a informação
     * a enviar para o servidor.
     */
    data = $("#registo").serialize();

    /*
     * Podemos também definir a variável data da seguinte forma:
     *
     * nome = 'nome=' + $('#nome').val();
     * password = 'password=' + $('#password').val();
     * email = 'email=' + $('#email').val();
     *
     * data = nome + '&' + password + '&' + email;
     */

    //Começa aqui o pedido ajax
    $.ajax({
        //Tipo do pedido que, neste caso, é POST
        type: 'POST',
        /*
         * URL do ficheiro que para o qual iremos enviar os dados.
         * Pode ser um url absoluto ou relativo.
         */
        url: 'processar.php',
        //Que dados vamos enviar? A variável "data"
        data: data,
        //O tipo da informação da resposta
        dataType: 'json'
    }).done(function(response) {

        /*
         * Quando a chamada Ajax é terminada com sucesso,
         * o javascript confirma o status da operação
         * com a variável que enviámos no formato json.
         */
        if(response.status) {
            //Se for positivo, mostra ao utilizador uma janela de sucesso.
            alert('Registo bem Sucedido!');
        } else {
            //Caso contrário dizemos que aconteceu algum erro.
            alert('Uups! Ocorreu algum erro!');
        }

    }).fail(function(xhr, desc, err) {
        /*
         * Caso haja algum erro na chamada Ajax,
         * o utilizador é alertado e serão enviados detalhes
         * para a consola javascript que pode ser visualizada
         * através das ferramentas de desenvolvedor do browser.
         */
        alert('Uups! Ocorreu algum erro!');
        console.log(xhr);
        console.log("Detalhes: " + desc + "nErro:" + err);
    });
}
```

Relembro que todo o código neste artigo é para fins demonstrativos e que para fins profissionais devem ser aplicadas medidas de segurança de forma a que terceiros não consigam obter os dados que estão a ser transmitidos.

Depois de guardar o ficheiro, podemos abrir o formulário, preenchê-lo, enviá-lo e ver se tudo foi bem sucedido.

Qualquer dúvida que tenham procurado e não tenham conseguido resolver, não hesitem em perguntar mas nunca se esqueçam: antes de **perguntar**, devem **procurar** e **tentar** porque só assim é que vão realmente compreender a 100% :)