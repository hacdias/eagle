---
description: O modelo MVC (Model-View-Controller) é um modelo extremamente utilizado nas aplicações web. Apresentamos uma breve explicação sobre este modelo.
publishDate: "2014-11-22T16:02:56.000Z"
tags:
- MVC
- PHP
- Programação
title: Modelo MVC - Uma breve explicação
---

A programação é algo fantástico que nos permite fazer qualquer coisa das mais diversas formas. Podemos criar, inovar, melhorar e até nos divertirmos. Existem várias formas de programar, várias maneiras.

Cada um, ao longo do tempo, vai adotando diversas formas de programar, diversas idiossincrasias que o vai distinguir ao longo do tempo. Mas o que vos trazemos hoje não é nenhuma idiossincrasia, é o modelo **MVC**.

<!--more-->

**MVC** é um modelo de arquitetura de software que é muito utilizado atualmente mas, por vezes, pode tornar-se confuso inicialmente (como me aconteceu) mas, depois de alguma pesquisa, cheguei à conclusão de que é muito simples **compreender** este modelo.

## O que quer dizer MVC?

Em primeiro lugar, é importante saber o que quer dizer MVC e o que é e para que serve cada uma das suas componentes.

**MVC** quer dizer, em inglês, _model-view-controller_ e, em português, podemos traduzir para **modelo-vista-controlador**. Estas são também as três componentes base deste modelo.


## Vista

{{< figure src="olho.jpg" title="As vistas são os olhos da aplicação" >}}

A camada **Vista** (_View_) é aquela que é mostrada ao utilizador, aquela que com a qual o utilizador vai interagir. É a **camada de apresentação**. A camada dos estilos, do _design_.

Geralmente, esta camada não conta com lógica de programação podendo, por vezes, ser "puro" HTML com alguns bocados de outra linguagem para, por exemplo, inserir algum dado necessário.


## Controladores

{{< figure src="cerebro.jpg" title="Os controladores, o cérebro" >}}

A segunda camada a ser mencionada é a dos **Controladores** (_Controllers_) e é nela que **a magia acontece**. É a camada intermédia do padrão MVC e toda a lógica está aqui contida.

Quando acedemos a um site cujo modelo de programação tenha sido MVC, automaticamente acedemos ao controlador que vai receber o nosso pedido. Logo de seguida, ele coordena todos os processos seguintes: pedir informação, receber informação, mostrar a página (**Vista**) ao utilizador, etc,

## Modelos

{{< figure src="sist-nev.jpg" title="E o sistema nervoso" >}}

Esta é a camada mais invisível ao utilizador. É nela que tudo o que tem haver com dados é feito: pedir coisas à base de dados, inserir coisas, eliminar coisas, trocar coisas, etc.

O controlador (mencionado acima), envia informação para o modelo armazenar/apagar/etc da base de dados. De forma generalizada, o **modelo** trabalha com os dados.


## É útil?


Diga-me você! Existem várias vantagens e desvantagens em trabalhar com o modelo MVC. Por um lado, é bom porque cada coisa está no seu devido sítio sendo mais fácil trabalhar e alterar estruturas.

Por outro lado, não é recomendado para pequenas aplicações visto que a sua complexidade pode prejudicar um pouco a performance e também o design. **Qual a sua opinião acerca do modelo MVC?**