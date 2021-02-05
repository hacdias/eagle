---
description: Os operadores '==' e '===' por vezes podem confundir-nos um pouco em relação ao seu uso e para que servem. Fica aqui a explicação de cada um."
publishDate: "2014-07-23T18:31:23.000Z"
tags:
- php
- javascript
title: Diferenças entre '==' e '===' em PHP
---

É habitual utilizarem-se diversos operadores quando o assunto é programação. Existem vários até: aritméticos, comparativos e outros.

Dentro de cada categoria existem vários sendo que a dos **comparativos** não é excepção e hoje trago-vos dois operadores comparativos, muito parecidos... ou talvez não: ```==``` e ```===```.

<!--more-->

Há já algum tempo que me questionava em relação ao uso de ambos os operadores pois não tinha a certeza de qual devia utilizar nas diversas situações que me apareciam.

Depois de uma pequena pesquisa, encontrei esta tabela na resposta a uma pergunta no [Stackoverflow](http://stackoverflow.com/) e tomei a liberdade de a transcrever para aqui:

|       ----- | **false** | **null** | **array()** | **0** | **"0"** | **0x0** | **"0x0"** | **"000"** | **"0000"** |
|-------------|-----------|----------|-------------|-------|---------|---------|-----------|-----------|------------|
| **false**   | === 		  | ==   | ==      | ==  | ==  | ==  | !=    | !=    | !=     |
| **null**    | ==  		  | ===  | ==      | ==  | !=  | ==  | !=    | !=    | !=     |
| **array()** | ==  		  | ==   | ===     | !=  | !=  | !=  | !=    | !=    | !=     |
| **0**       | ==  		  | ==   | !=      | === | ==  | === | ==    | ==    | ==     |
| **"0"**     | ==  		  | !=   | !=      | ==  | === | ==  | ==    | ==    | ==     |
| **0x0**     | ==    		| ==   | !=      | === | ==  | === | ==    | ==    | ==     |
| **"0x0"**   | !=   			| !=   | !=      | ==  | ==  | ==  | ===   | ==    | ==     |
| **"000"**   | !=   			| !=   | !=      | ==  | ==  | ==  | ==    | ===   | ==     |
| **"0000"**  | !=   			| !=   | !=      | ==  | ==  | ==  | ==    | =     | ===    |

Como assim? Essa tabela compara diversas possibilidades. `!=` quer dizer que é diferente, e os outros dois? Aqui está a diferença entre `==` e `===`:

  * `==` compara os valores das variáveis ou seja, a **igualdade**.
  * `===` compara os valores das variáveis **e** o tipo das mesmas ou seja, a **identidade**.

Estes dois operadores têm dois opostos: o operador ```!=``` e o operador ```!==``` que comparam as mesmas coisas ou seja, o primeiro confirma se as variáveis **não têm valores iguais** e o segundo verifica se **não têm os valores, nem o tipo iguais**.


## Exemplo


Assim podemos dizer, por exemplo,  que a comparação ```"1" == 1```  retorna ```true``` mas que ```"1" === 1```  retorna ```false```.

Na primeira comparação só são comparados os valores das variáveis logo 1 é, realmente, igual a 1 daí retornar verdadeiro.

Na segunda, os tipos também são comparados, enquanto que a primeira é uma string , a segunda é do tipo int  logo é falso apesar do conteúdo ser igual.



* * *



Esta explicação é equivalente para outras linguagens de programação como **JavaScript**, por exemplo.

Agora que já sei a diferença entre estes dois operadores, é hora de ir mudar alguns "==" e "===".  Podes ler mais sobre operadores comparativos em PHP [aqui](http://au.php.net/manual/en/language.operators.comparison.php).