/**
 * 全站顶栏：根据 Session 显示「登录/注册」或用户名
 * 页面需包含 .site-header 结构
 */
(function ($) {
    function bindSiteHeader() {
        var $root = $(".site-header");
        if (!$root.length) {
            return;
        }
        $.get("/api/v1.0/session", function (resp) {
            if ("0" == resp.errno && resp.data && resp.data.name) {
                $root.find(".user-info .user-name").text(resp.data.name);
                $root.find(".user-info").show();
                $root.find(".register-login").hide();
                $(".quick-nav").css("display", "flex");
            } else {
                $root.find(".register-login").show();
                $root.find(".user-info").hide();
            }
        }, "json").fail(function () {
            $root.find(".register-login").show();
            $root.find(".user-info").hide();
        });
    }

    $(document).ready(bindSiteHeader);
})(window.jQuery);
