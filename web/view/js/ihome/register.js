var imageCodeId = "";

function generateUUID() {
    var d = new Date().getTime();
    if(window.performance && typeof window.performance.now === "function"){
        d += performance.now();
    }
    var uuid = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        var r = (d + Math.random()*16)%16 | 0;
        d = Math.floor(d/16);
        return (c=='x' ? r : (r&0x3|0x8)).toString(16);
    });
    return uuid;
}

// 生成一个图片验证码的编号，并设置页面中图片验证码img标签的src属性
function generateImageCode() {
    imageCodeId = generateUUID();
    var imageCodeUrl = "/api/v1.0/imagecode/" + imageCodeId;
    $(".image-code>img").attr("src", imageCodeUrl);
}

$(document).ready(function() {
    generateImageCode();
    $("#mobile").focus(function(){
        $("#mobile-err").hide();
    });
    $("#imagecode").focus(function(){
        $("#image-code-err").hide();
    });
    $("#password").focus(function(){
        $("#password-err").hide();
        $("#password2-err").hide();
    });
    $("#password2").focus(function(){
        $("#password2-err").hide();
    });
    $(".form-register").submit(function(e){
        e.preventDefault();
        var mobile = $("#mobile").val();
        var imagecode = $("#imagecode").val();
        var passwd = $("#password").val();
        var passwd2 = $("#password2").val();
        if (!mobile || !/^\d{11}$/.test(mobile)) {
            $("#mobile-err span").html("请填写正确的手机号！");
            $("#mobile-err").show();
            return;
        }
        if (!imagecode) {
            $("#image-code-err span").html("请填写图片验证码！");
            $("#image-code-err").show();
            return;
        }
        if (!passwd) {
            $("#password-err span").html("请填写密码!");
            $("#password-err").show();
            return;
        }
        if (passwd != passwd2) {
            $("#password2-err span").html("两次密码不一致!");
            $("#password2-err").show();
            return;
        }

        var req = {
            "mobile": mobile,
            "password": passwd,
            "imagecode": imagecode,
            "uuid": imageCodeId
        };

        $.ajax({
            url: "/api/v1.0/users",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify(req),
            dataType: "json",
            success: function(resp){
                if (resp.errno == "0") {
                    // 注册成功并已写入 Session，跳转首页
                    location.href = "/home/index.html";
                    return;
                }
                // 验证码错误等提示在图验区域；其它错误在确认密码下方
                if (resp.errmsg && resp.errmsg.indexOf("验证码") !== -1) {
                    $("#image-code-err span").html(resp.errmsg);
                    $("#image-code-err").show();
                    generateImageCode();
                    $("#imagecode").val("");
                } else {
                    $("#password2-err span").html(resp.errmsg || "注册失败");
                    $("#password2-err").show();
                    generateImageCode();
                    $("#imagecode").val("");
                }
            },
            error: function () {
                $("#password2-err span").html("网络或服务异常，请稍后重试");
                $("#password2-err").show();
                generateImageCode();
            }
        });
    });
})
