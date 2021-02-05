---
description: Continuamos a construir a nossa framework que terá como base o modelo MVC. Neste artigo criamos o coração da nossa framework.
publishDate: "2014-12-19T12:26:59.000Z"
tags:
- php
title: MVC na linguagem PHP - II
---

A pedido de um leitor, decidimos escrever um artigo em que exemplificamos este maravilhoso (ou não) modelo MVC utilizando a linguagem de programação PHP.

Depois de publicarmos a [primeira parte](/articles/2014/12/mvc-linguagem-php/), onde explicámos a estrutura da pequena _framework_ que iremos criar, aqui está a segunda parte.

<!--more-->

Hoje iremos trabalhar no coração da aplicação, ou seja, na diretoria `app_core/libs`. Nesta pasta mãe, irão ser colocados todos os ficheiros PHP que serão as classes mãe de todos os outros ficheiros.

## **Bootstrap.php**

Vamos começar por criar um ficheiro denominado Bootstrap.php onde iremos colocar todo o seguinte código:

```php
<?php

/**
 * Class Bootstrap
 *
 * @package MVC PHP Bootstrap
 */
class Bootstrap
{
    private $_url = null;
    private $_controller = null;

    private $_errorFile = 'error.php';

    /**
     * Starts the Bootstrap
     *
     * @return boolean
     */
    public function init()
    {
        $this->_getUrl();

        if (empty($this->_url[0])) {
            $this->_url[0] = 'index';
        }

        $this->_controller();
        $this->_method();

        return false;
    }

    /**
     * This function get the content of 'url' variable
     * of HTTP GET method. See the .htaccess for more
     * information.
     */
    private function _getUrl()
    {
        $url = isset($_GET['url']) ? $_GET['url'] : null;
        $url = rtrim($url, '/');
        $url = filter_var($url, FILTER_SANITIZE_URL);
        $this->_url = explode('/', $url);
    }

    /**
     * This function initializes the controller that
     * matches with the current url.
     *
     * @return bool
     */
    private function _controller()
    {
        $file = DIR_CONTROLLERS . $this->_url[0] . '.php';

        if (file_exists($file)) {
            require $file;

            $controller = "Controller" . $this->_url[0];

            $this->_controller = new $controller($this->_url[0]);

            return false;
        } else {
            $this->_error();
            return false;
        }

    }

    /**
     * This function calls the method depending on the
     * url fetched above.
     */
    private function _method()
    {
        $length = count($this->_url);

        if ($length > 1) {
            if (!method_exists($this->_controller, $this->_url[1])) {
                $this->_error();
            }
        }

        switch ($length) {
            case 5:
                //Controller->Method(Param1, Param2, Param3)
                $this->_controller->{$this->_url[1]}($this->_url[2], $this->_url[3], $this->_url[4]);
                break;

            case 4:
                //Controller->Method(Param1, Param2)
                $this->_controller->{$this->_url[1]}($this->_url[2], $this->_url[3]);
                break;

            case 3:
                //Controller->Method(Param1, Param2)
                $this->_controller->{$this->_url[1]}($this->_url[2]);
                break;

            case 2:
                //Controller->Method(Param1, Param2)
                $this->_controller->{$this->_url[1]}();
                break;

            default:
                $this->_controller->index();
                break;
        }
    }

    /**
     * Display an error page if there's no controller
     * that corresponds with the current url.
     */
    private function _error()
    {
        require DIR_CONTROLLERS . $this->_errorFile;

        $this->_controller = new ControllerError();
        $this->_controller->index();

        exit;
    }

}
```

O código acima é aquele que irá inicializar todo a aplicação. Em primeiro lugar, gostava de dizer que a estrutura do URL da aplicação será a seguinte:

```bash
http://site/controlador/método/arg1/arg2/[arg...]
```

Vamos então ver, por partes, o que faz cada uma das funções declaradas acima.

### `init`

A função `init` é a função onde tudo começa. Em primeiro lugar, esta função chama a função `_getUrl` que recebe o URL atual (será analisada mais à frente).

De seguida, esta função define a página como index caso nenhuma página esteja definida no URL. Então a função chama duas outras que iremos ver de seguida.

### `_getUrl`

Esta função recebe a variável url que foi passada através do método GET (em breve veremos as modificações que têm que ser efetuadas no `.htaccess` para que seja passada esta variável).

Aqui é utilizado o [operador ternário](/articles/2014/07/operador-ternario/) de forma a que a variável `$url` seja igual a null  caso não haja nenhum conteúdo na variável url que foi passada através do método GET.

Depois é removida a última barra (/) da variável com a função `rtrim`. De seguida é aplicado um filtro à array  de forma a remover todos os caracteres não permitidos aqui.

Finalmente, a variável do url da classe (`$_url`) é igualada à "explosão" da variável `$url` que se irá tornar num array .

### `_controller`

Esta função, em primeiro lugar, define o caminho do ficheiro do controlador correspondente ao URL atual. O caminho será igual à constante `DIR_CONTROLLERS` + a primeira parte do url + a extensão do ficheiro que é `.php`.

De seguida, é feita a verificação se o controlador em questão existe. Se existir, o controlador é inicializado, caso contrário, o fluxo da aplicação é direcionado para a função de erro (`_error`).


### `_method`


Esta função é como um GPS: ela é que envia o fluxo para o sítio correto. Isto vai ser feito dependendo do que foi enviado no URL.

Esta função irá executar o método em questão que, caso não seja especificado nenhum, é o método index do controlador em questão.


### `_error`

Finalmente, temos a função de erro que irá inicializar o controlador dos erros. O ficheiro que corresponde a este controlador é definido na variável `$_errorFile`, que eu coloquei `error.php`.

## `Controller.php`

De momento, o que foi feito até agora pode aparentar não ter muito sentido, mas com o encaixar das peças tudo irá ser mais claro. Vejamos agora a classe mãe dos Controladores.

```php
<?php

/**
 * Class Controller
 *
 * @package MVC PHP Bootstrap
 */
class Controller
{
    /**
     * The constructor of this class automatically initializes
     * the View and sets the corresponding model path. If the
     * model file exists, it calls it.
     *
     * @param $name
     */
    function __construct($name)
    {
        $this->view = new View();

        $path = ROOT . 'models/' . $name . '.php';

        if (file_exists($path)) {
            require $path;

            $modelName = "Model" . $name;
            $this->model = new $modelName();
        }
    }

}
```

Esta classe é claramente menos complexa que a anterior e conta apenas com o seu construtor que recebe o nome do controlador e inicializa, automaticamente, a *View*.

De seguida, o construtor constrói o caminho até ao modelo do controlador em questão, que caso o controlador se chame "about" , o caminho para o modelo seria `ROOT . 'models/about.php'`.

Depois é feita a verificação se existe o ficheiro do modelo e, caso este exista, é inicializado o modelo do controlador.

Mas, porque é que esta verificação é feita? Porque nem todas as páginas utilizarão a base de dados. Páginas estáticas como, por exemplo, a página sobre, não necessitam, geralmente, de manipulação de dados.

## `Database.php`

A classe Database vai estar intimamente ligada com a classe PDO e será com ela que inicializaremos a ligação à base de dados e não com a PDO, pois a Database  é baseada na PDO. Ora veja:

```php
<?php

/**
 * Class Database
 *
 * @package MVC PHP Bootstrap
 */
class Database extends PDO
{

    public function __construct($DB_TYPE, $DB_HOST, $DB_NAME, $DB_USER, $DB_PASS)
    {
        parent::__construct($DB_TYPE . ':host=' . $DB_HOST . ';dbname=' . $DB_NAME, $DB_USER, $DB_PASS);
        $this->exec("SET NAMES 'utf8';");
    }

    /**
     * Function used to select something of the database.
     *
     * @param string $sql An SQL string
     * @param array $array Parameters to bind
     * @param const|int $fetchMode A PDO Fetch mode
     * @return mixed
     */
    public function select($sql, $array = array(), $fetchMode = PDO::FETCH_ASSOC)
    {
        $sth = $this->prepare($sql);

        foreach ($array as $key => $value) {
            $sth->bindValue("$key", $value);
        }

        $sth->execute();
        return $sth->fetchAll($fetchMode);
    }

    /**
     * Function used to insert things in the database.
     *
     * @param string $table A name of table to insert into
     * @param string $data An associative array
     */
    public function insert($table, $data)
    {
        ksort($data);

        $fieldNames = implode('`, `', array_keys($data));
        $fieldValues = ':' . implode(', :', array_keys($data));

        $sth = $this->prepare("INSERT INTO $table (`$fieldNames`) VALUES ($fieldValues)");

        foreach ($data as $key => $value) {
            $sth->bindValue(":$key", $value);
        }

        $sth->execute();
    }

    /**
     * Function used to update things on the database.
     *
     * @param string $table A name of table to insert into
     * @param string $data An associative array
     * @param string $where the WHERE query part
     */
    public function update($table, $data, $where)
    {
        ksort($data);
        $fieldDetails = NULL;

        foreach ($data as $key => $value) {
            $fieldDetails .= "`$key`=:$key,";
        }

        $fieldDetails = rtrim($fieldDetails, ',');

        $sth = $this->prepare("UPDATE $table SET $fieldDetails WHERE $where");

        foreach ($data as $key => $value) {
            $sth->bindValue(":$key", $value);
        }

        $sth->execute();

    }

    /**
     * Function used to delete things from the database.
     *
     * @param string $table
     * @param string $where
     * @param integer $limit
     * @return integer Affected Rows
     */
    public function delete($table, $where, $limit = 1)
    {
        return $this->exec("DELETE FROM $table WHERE $where LIMIT $limit");
    }

}
```

Não há muito a falar sobre esta classe visto que ela contém apenas algumas funções para agilizar diversas operações como inserções na base de dados, seleções, atualizações e eliminações.

Acrescento que o construtor deve receber todos aqueles itens que escrevemos nas constantes da configuração: todos os dados relativos à conexão à Base de Dados.


## `Model.php`

Vejamos então a classe superior de todos os modelos que irão constituir a nossa aplicação:

```php
<?php

/**
 * Class Model
 *
 * @package MVC PHP Bootstrap
 */
class Model
{
    /**
     * The constructor of this class automatically initializes
     * the Database.
     */
    function __construct()
    {
        $this->db = new Database(DB_TYPE, DB_HOST, DB_NAME, DB_USER, DB_PASS);
    }

}
```


Como pode ver, a classe é pequena e o seu construtor apenas inicializa uma instância da conexão à Base de Dados que será utilizada em todos os modelos.


## ```View.php```

O modelo da *View* que já foi muito falado acima também é muito pequeno.

```php
<?php

/**
 * Class View
 *
 * @package MVC PHP Bootstrap
 */
class View
{
    protected $_data;

    function __construct()
    {
        //Views Contruct
    }

    public function render($name)
    {
        require DIR_VIEWS . 'header.php';
        require DIR_VIEWS . $name . '.php';
        require DIR_VIEWS . 'footer.php';

    }

    public function setData($data)
    {
        $this->_data = $data;
    }

}
```

Neste é inicializada uma variável chamada `$_data` que irá conter todo o conteúdo que deverá ser enviado para o HTML de forma a ser imprimido.

Temos também a função render  que é aquela que vai incluir (ou requerer) todos os ficheiros para serem apresentados. Dividi os ficheiros em três partes: o cabeçalho (_header_), o principal e o rodapé (_footer_).

Temos, por fim, a função setData que será utilizada para definir a variável da vista. Utilizei uma função para não haver manipulação direta das variáveis da classe.

> Provavelmente reparou que coloquei vários comentário em Inglês. No final desta saga de artigos vou colocar esta *framework* simples no GitHub de forma a que todos os que queiram possam contribuir ou até mesmo utilizar. :)

Até ao próximo artigo. O principal já está feito. Faltam os dois ficheiros que iniciarão tudo (e mais algumas coisinhas).