<#--
  crashr's frame for every sign-in page. keycloak.v2's pages call registrationLayout and fill its sections: "header"
  (the title), "form", "socialProviders" and "info". The frame draws them in one card under the crashr mark, on the
  app's dark ground. A section the page leaves empty draws nothing. Under it, the languages the realm offers.
-->
<#import "footer.ftl" as loginFooter>

<#-- Whether a section holds anything: keycloak.v2 prints a comment naming the template in every section. -->
<#function hasContent value>
    <#local text = value?is_markup_output?then(value?markup_string, value?string)>
    <#return text?replace("<!--[\\s\\S]*?-->", "", "r")?trim?has_content>
</#function>

<#macro registrationLayout bodyClass="" displayInfo=false displayMessage=true displayRequiredFields=false extra...>
<#local header><#nested "header"></#local>
<#local social><#nested "socialProviders"></#local>
<#local info><#if displayInfo><#nested "info"></#if></#local>
<!DOCTYPE html>
<html class="${properties.kcHtmlClass!}" lang="${lang}">
<head>
    <meta charset="utf-8">
    <meta name="color-scheme" content="dark">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="robots" content="noindex, nofollow">
    <title>${msg("loginTitle", "crashr")}</title>
    <link rel="icon" href="${url.resourcesPath}/img/favicon.svg" type="image/svg+xml">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link href="https://fonts.googleapis.com/css2?family=Nunito:wght@500;600;700;800&display=swap" rel="stylesheet">
    <#list (properties.styles!"")?split(' ') as style><#if style?has_content>
    <link href="${url.resourcesPath}/${style}" rel="stylesheet">
    </#if></#list>
    <script type="importmap">{ "imports": { "rfc4648": "${url.resourcesCommonPath}/vendor/rfc4648/rfc4648.js" } }</script>
    <#if scripts??><#list scripts as script><script src="${script}" type="text/javascript"></script></#list></#if>
    <script type="module" src="${url.resourcesPath}/js/passwordVisibility.js"></script>
    <script type="module">
        import { startSessionPolling } from "${url.resourcesPath}/js/authChecker.js";
        startSessionPolling("${url.ssoLoginInOtherTabsUrl?no_esc}");
    </script>
    <#if authenticationSession??>
    <script type="module">
        import { checkAuthSession } from "${url.resourcesPath}/js/authChecker.js";
        checkAuthSession("${authenticationSession.authSessionIdHash}");
    </script>
    </#if>
</head>
<body class="${properties.kcBodyClass!} ${bodyClass}" data-page-id="login-${pageId}">
<div class="h-page">
    <main class="h-card" aria-labelledby="kc-page-title">
        <header class="h-brand">
            <svg width="32" height="32" viewBox="0 0 28 28" aria-hidden="true"><circle cx="14" cy="9" r="6" fill="#2A4BFF"/><path d="M3 22q11-9 22 0" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"/></svg>
            <span class="h-brand-name">crashr<span class="h-brand-tld">.xyz</span></span>
        </header>
        <#if hasContent(header)>
            <h1 id="kc-page-title" class="h-title">${header}</h1>
        </#if>

        <#if auth?has_content && auth.showUsername() && !auth.showResetCredentials()>
            <div class="h-who">
                <span class="h-who-value" id="kc-attempted-username">${auth.attemptedUsername}</span>
                <a id="reset-login" class="h-link" href="${url.loginRestartFlowUrl}">${msg("restartLoginTooltip")}</a>
            </div>
        </#if>

        <#if displayRequiredFields>
            <p class="h-hint"><span class="h-required" aria-hidden="true">*</span> ${msg("requiredFields")}</p>
        </#if>

        <#if displayMessage && message?has_content && (message.type != 'warning' || !isAppInitiatedAction??)>
            <div class="h-alert h-alert-${message.type}" role="${(message.type = 'error')?then('alert', 'status')}">
                <span class="h-alert-text kc-feedback-text">${kcSanitize(message.summary)?no_esc}</span>
            </div>
        </#if>

        <#nested "form">

        <#if auth?has_content && auth.showTryAnotherWayLink()>
            <form id="kc-select-try-another-way-form" class="h-form" action="${url.loginAction}" method="post">
                <input type="hidden" name="tryAnotherWay" value="on"/>
                <button id="try-another-way" type="submit" class="h-btn h-btn-secondary h-btn-block">${msg("doTryAnotherWay")}</button>
            </form>
        </#if>

        <#if hasContent(social)><div class="h-social">${social}</div></#if>
        <#if hasContent(info)><div id="kc-info" class="h-info">${info}</div></#if>
    </main>
    <p class="h-tagline">${msg("crashrTagline")}</p>
    <#if realm.internationalizationEnabled && locale.supported?size gt 1>
        <nav class="h-langs" aria-label="${msg("crashrLanguages")}">
            <#list locale.supported as language>
                <a class="h-lang" href="${language.url}" lang="${language.languageTag}"<#if language.languageTag == locale.currentLanguageTag> aria-current="true"</#if>>${language.languageTag?upper_case}</a>
            </#list>
        </nav>
    </#if>
</div>
</body>
</html>
</#macro>
