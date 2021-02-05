---
description: O Git é dos sistemas de controlo de versão mais utilizados por todo o mundo. Hoje é hora de falar um pouco da história do Git e do GitHub."
publishDate: "2014-09-02T14:09:59.000Z"
tags:
- git
title: Git - Uma história e repositórios do GitHub"
---

**Git** é um sistema de controlo de versão e gestão de código fonte extremamente utilizado em todos os cantos do mundo.

Hoje vamos aprender a criar um simples repositório no GitHub e fazer a primeira "commit" ou seja, enviar os primeiros dados para o repositório.

<!--more-->

## Um pouco de história...

O kernel (ou núcleo) do Linux é um projeto de código aberto com um tamanho muito grande. No período de 1991 a 2002 houveram grandes mudanças no kernel do Linux visto que foi o período onde a manutenção foi maior.

Em 2002, este projeto começou a utilizar um sistema de controlo de versão proprietário, o [BitKeeper](http://www.bitkeeper.com/).

Três anos mais tarde, em 2015, o relacionamento existente entre a comunidade que desenvolvia o Kernel e a empresa do BitKeeper desfez-se e isso levou a comunidade a pensar numa nova solução.

A comunidade começou então a desenvolver um novo sistema sendo que os objetivos que pretendiam atingir com este sistema eram a **velocidade**, **um software robusto, design simples** e capaz de lidar com **grandes projetos**.

Foi assim que nasceu o Git. Ao longo dos anos tem vindo a tornar-se um sistema de controlo de versão mundialmente conhecido e utilizado.


## Porquê o GitHub?

O tutorial de hoje é, em parte para o GitHub, mas a restante parte é indiferente pois basta ser um repositório Git.

Estamos a utilizar o GitHub por ser um serviço muito conhecido e utilizado e a criação de repositórios ser simples, fácil e rápida.

## _Let's start_

Os requisitos são ter uma ligação à Internet, ter o Git instalado no vosso computador [através desta página](http://git-scm.com/) e ter uma conta no [GitHub](https://github.com).

Assim, começamos por ir ao site do GitHub e clicamos no botão **+ → New Repository** como podem ver na imagem abaixo.

{{< figure
    src="github01.jpg" >}}

Depois tens que preencher o nome do repositório, a descrição e, por agora, não são precisas mais configurações. No final é só clicar no botão verde.

{{< figure
    src="github02.jpg" >}}

Copia o código HTTPS ou SSH utilizando o botão que aparece do lado direito.

{{< figure
    src="github03.jpg" >}}

Agora que já temos o repositório configurado no GitHub, vamos colocá-lo no nosso computador. Esta parte agora **é igual independentemente do serviço utilizado**, GitHub, Bitbucket, etc, basta ser Git.

## Git, git, git e mais git

Começa por abrir a linha de comandos/shell ou qualquer outra coisa que use o teu sistema operativo no local onde queres que o repositório seja clonado/copiado/armazenado.

Tem em atenção que, se abrires na pasta ```D:\dev```, o repositório vai ser clonado em ```D:\dev\nomeDoRepo```. Executa então o seguinte comando:

```bash
git clone <URL-QUE-COPIASTE>
```

Agora é só esperar uns segundos até que o repositório seja clonado. Deves receber uma mensagem semelhante à seguinte:

```bash
Cloning into 'RepositorioDeExemploCOXPE'...
warning: You appear to have cloned an empty repository.
Checking connectivity... done.
```

Agora naveguem até à pasta do repositório utilizando o comando cd <NOME-DO-REPO> . Agora que estão dentro da pasta, devem ver uma pasta chamada ```.git``` oculta.

Vamos então criar o nosso primeiro ficheiro para enviar para o repositório. Comecem por executar o seguinte comando:

```bash
echo Este é o meu primeiro repositório > README.md
```

Esse comando vai fazer com que seja criado um ficheiro ```README.md``` com o conteúdo "Este é o meu primeiro repositório" (sem aspas). Agora é hora de enviar este ficheiro para o servidor. Para isso executamos um dos seguintes comandos:

```bash
#Para adicionar apenas UM ficheiro
git add <nome-do-ficheiro>

#Para adicionar todas as novas modificações
git add -all
```

Agora temos que adicionar uma mensagem que vai identificar este envio. Para isso escreve o seguinte, substituindo "My Message" pelo que quiseres (mas sempre com aspas):

```bash
git commit -m "My Message"
```

Agora é hora de enviar as mudanças para o repositório remoto. Para isso executa o seguinte comando que irá enviar as alterações para a "branch" master:

```bash
git push origin

#Depois insere os dados de utilizador quando pedido
```

## Pronto...

Pronto, já está! Se fores agora ao repositório que criaste irás ver o novo ficheiro. Como o ficheiro foi denominado ```README.md```, este estará sempre visível quando abrimos o repositório na parte abaixo da listagem dos ficheiros.

Espero que tenham gostado deste pequeno tutorial :)


> Fiquem atentos. Em breve teremos novidades com o [Pplware](http://pplware.com):)