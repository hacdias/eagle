---
description: Explicamos como fazer a listagem em log de todos os erros produzidos pela linguagem PHP num ficheiro através do .htaccess.
publishDate: "2014-08-14T10:41:38.000Z"
tags:
- php
title: Guardar os erros do PHP num ficheiro com o .htaccess
---

Quando os programadores de soluções web estão a programar alguma aplicação é normal que possam surgir erros durante o desenvolvimento e a linguagem de programação, por norma, mostra-os no browser.

Mesmo depois do desenvolvimento concluído e do produto lançado, podem surgir erros porém é pouco profissional mostrar os erros "à paisana" no browser do utilizador sendo até uma boa "ajuda" para hackers.

<!--more-->

Hoje vamos explicar como podemos utilizar o ficheiro `.htaccess` para fazer o registo em log (_logging_) dos erros dados pela linguagem de programação PHP num ficheiro.

Como devem ter previsto, o único ficheiro que é necessário modificar é o `.htaccess`. Vamos começar por definir algumas `php_flags` de forma a não mostrar nenhum erro ao utilizador.

```apache
# Não mostra erros de inicialização
php_flag display_startup_errors off
# Não mostra os restantes erros
php_flag display_errors off
# Não mostra erros de markup HTML
php_flag html_errors off
```

O código acima, e o restante, está comentado para saber o que faz cada linha. Agora, depois de termos desativado os erros "ao público", vamos fazer com que estes sejam guardados num ficheiro.

```apache
# Ativa o registo em log dos erros
php_flag log_errors on
# Desativa a ignoração a erros repetidos
php_flag ignore_repeated_errors off
# Desativa "Ignorar erros de fonte única"
php_flag ignore_repeated_source off
# Ativa log de vazamentos de memória do php
php_flag report_memleaks on
# Preserva os erros mais recentes
php_flag track_errors on
# Desativa a formatação de erros com links de referência
php_value docref_root 0
php_value docref_ext 0
# Especifica um caminho para o ficheiro de log
php_value error_log /home/error.log

# Especifica para guardar todos os erros
php_value error_reporting -1
# Desativa o tamanho máximo de erros
php_value log_errors_max_len 0
```

Agora os erros já estão a ser guardados num ficheiro de log, mas continua a existir um problema. Se acedermos ao URL onde está guardado, vamos poder aceder ao ficheiro.

Isto pode ser utilizado como arma por qualquer pessoa, nomeadamente por _hackers_. Vamos então proteger o ficheiro de forma a que o público não lhe tenha acesso.

```apache
# Proteger o ficheiro
<Files error.log>
 Order allow,deny
 Deny from all
 Satisfy All
</Files>
```

Não se esqueçam de colocar o nome do ficheiro correto substituindo `error.log` pelo nome que querem. Alerto também para alterarem o caminho `/home/error.log` para o caminho em questão. Aqui está o código completo sem qualquer comentário, excelente para ser utilizado:

```apache
php_flag display_startup_errors off
php_flag display_errors off
php_flag html_errors off
php_flag log_errors on
php_flag ignore_repeated_errors off
php_flag ignore_repeated_source off
php_flag report_memleaks on
php_flag track_errors on
php_value docref_root 0
php_value docref_ext 0
php_value error_log /home/error.log
php_value error_reporting -1
php_value log_errors_max_len 0

<Files error.log>
 Order allow,deny
 Deny from all
 Satisfy All
</Files>
```

Depois de fazerem as edições no caminho e nome do ficheiro, basta guardarem o vosso .htaccess  e verificam que os erros produzidos pelo PHP irão ser salvos no ficheiro em questão.

Acrescento que, a qualquer momento, pode utilizar a função ```error_log('Erro aqui')``` do PHP para enviar um erro para o log.

Espero que este artigo vos tenha ajudado :)

**Edição:** Tal como o leitor António mencionou, isto apenas funciona caso o PHP tenha o [módulo do Apache ativo. ](http://support.tigertech.net/php-value)