---
description: Depois de termos ensinado a configurar o plugin php-gettext com o PHP, agora é hora de saber como utilizar o plugin.
publishDate: "2014-09-28T15:06:47.000Z"
tags:
- php
title: PHP - Como utilizar o Gettext para traduções [Parte 2]
---

Na [primeira parte](/articles/2014/09/php-como-utilizar-o-gettext-para-traducoes/) deste artigo, ensinámos a configurar o Gettext em conjunto com o PHP de forma a que hoje possamos criar as nossas primeiras traduções.

Agora é hora de criarmos as nossas primeiras traduções. Para isso, podemos instalar um programa chamado **Poedit** que nos vai ajudar. Podem descarregá-lo [aqui](http://poedit.net/).

<!--more-->

Abra o Poedit e clique em **Ficheiro → Novo** e selecione a língua **mãe** do seu site. No nosso caso, é português de Portugal (pt-PT). Depois disso, guarde o ficheiro em `lang/site_multi_lingua.pot`.

Logo de seguida, clique em **Extrair das fontes**. Agora, adicione o caminho do seu site na seção **Caminhos** da aba **Caminhos das fontes**.

De seguida, dirija-se à aba **Propriedades da tradução** e dê um nome ao projeto.Pode ainda escolher outras opções.

Na última aba, **Palavras-chave das fontes**, adicione `__`   e `_e`. Depois clique em **OK** e espere que os ficheiros sejam analisados.

Caso ocorra algum erro relacionado com caracteres não ASCII, clique em **Catálogo → Propriedades → Codificação do código fonte** e selecione UTF-8. Depois clique em **Atualizar.**

Agora deverá ver um ecrã semelhante ao seguinte:

{{< figure
    src="poedit00.jpg"
    caption="Poedit" >}}

Guarde este ficheiro e depois vá a **Novo → Novo de ficheiro POT/PO**. Selecione o ficheiro base que gravou antes e escolha a nova língua.

Agora basta clicar no item a traduzir e depois escrever a tradução na caixa de texto na parte de baixo do programa.

Depois, grave o ficheiro em `lang/en_GB/LC_MESSAGES/site_multi_lingua.po`. Altere `en_GB` pela língua que criou.

Agora, para testar se está tudo a funcionar, basta digital no URL `/?lang=en_GB` ou outra que queira testar.

Pode ver [neste repositório do GitHub](https://github.com/hacdias/labs/tree/master/php/multi-lang) todo o código que foi aqui produzido com alguns exemplos de linguagem adicionais.

> Refiro ainda que o Gettext faz _caching _de todas as _strings_.