function hrefBack() {
    history.go(-1);
}

function decodeQuery(){
    var search = decodeURI(document.location.search);
    return search.replace(/(^\?)/, '').split('&').reduce(function(result, item){
        var values = item.split('=');
        result[values[0]] = values[1];
        return result;
    }, {});
}

function showErrorMsg(msg) {
    if (msg) {
        $('.popup p').text(msg);
    }
    $('.popup_con').fadeIn('fast', function() {
        setTimeout(function(){
            $('.popup_con').fadeOut('fast', function(){});
        }, 1200);
    });
}

function calcNights(startDate, endDate) {
    if (!startDate || !endDate) {
        return 0;
    }
    var sd = new Date(startDate.replace(/-/g, '/'));
    var ed = new Date(endDate.replace(/-/g, '/'));
    if (isNaN(sd.getTime()) || isNaN(ed.getTime())) {
        return 0;
    }
    return Math.round((ed - sd) / (1000 * 3600 * 24));
}

function setSubmitting(on) {
    var $btn = $(".submit-btn");
    if (on) {
        $btn.addClass("disabled").text("提交中...");
        $btn.data("submitting", true);
    } else {
        $btn.removeClass("disabled").text("提交订单");
        $btn.data("submitting", false);
    }
}

$(document).ready(function(){
    var queryData = decodeQuery();
    var houseId = queryData["hid"];
    var housePrice = 0;
    var minDays = 1;
    var maxDays = 0;
    var houseOwnerId = 0;
    var currentUserId = 0;
    var canBook = false;

    if (!houseId) {
        showErrorMsg("缺少房源信息，请返回重试");
        setTimeout(function(){ location.href = "/home/index.html"; }, 1200);
        return;
    }

    // 判断用户是否登录
    $.get("/api/v1.0/session", function(resp) {
        if ("0" != resp.errno) {
            location.href = "/home/login.html";
        }
    }, "json");

    $(".input-daterange").datepicker({
        format: "yyyy-mm-dd",
        startDate: "today",
        language: "zh-CN",
        autoclose: true
    });

    function refreshAmount() {
        var startDate = $("#start-date").val();
        var endDate = $("#end-date").val();
        $(".order-amount>span").html("");

        if (!startDate || !endDate) {
            return;
        }
        if (startDate >= endDate) {
            showErrorMsg("离店日期须晚于入住日期");
            return;
        }
        var days = calcNights(startDate, endDate);
        if (days < 1) {
            showErrorMsg("日期有误，请重新选择");
            return;
        }
        if (minDays > 0 && days < minDays) {
            showErrorMsg("最少入住" + minDays + "晚");
            return;
        }
        if (maxDays > 0 && days > maxDays) {
            showErrorMsg("最多入住" + maxDays + "晚");
            return;
        }
        var amount = days * housePrice;
        $(".order-amount>span").html(amount.toFixed(2) + "(共" + days + "晚)");
    }

    $(".input-daterange").on("changeDate", refreshAmount);

    // 获取房屋的基本信息
    $.get("/api/v1.0/houses/" + houseId, function(resp){
        if ("0" != resp.errno && 0 != resp.errno) {
            showErrorMsg((resp && resp.errmsg) ? resp.errmsg : "房屋加载失败");
            return;
        }
        var house = resp.data.house;
        var imgUrls = house.img_urls || [];
        if (imgUrls.length > 0) {
            $(".house-info>img").attr("src", imgUrls[0]);
        }
        $(".house-text>h3").html(house.title || "");
        housePrice = Number(house.price || 0);
        $(".house-text>p>span").html(housePrice.toFixed(0));
        minDays = Number(house.min_days || 1);
        maxDays = Number(house.max_days || 0);
        houseOwnerId = Number(house.user_id || 0);
        currentUserId = Number(resp.data.user_id || 0);

        if (currentUserId && houseOwnerId && currentUserId === houseOwnerId) {
            canBook = false;
            setSubmitting(true);
            $(".submit-btn").text("不能预订自己的房源");
            showErrorMsg("不能预订自己的房源");
            return;
        }
        canBook = true;
        refreshAmount();
    }).fail(function(){
        showErrorMsg("房屋加载失败，请稍后重试");
    });

    // 订单提交
    $(".submit-btn").on("click", function() {
        if (!canBook || $(this).data("submitting")) {
            return;
        }
        if (currentUserId && houseOwnerId && currentUserId === houseOwnerId) {
            showErrorMsg("不能预订自己的房源");
            return;
        }

        var startDate = $("#start-date").val();
        var endDate = $("#end-date").val();
        if (!startDate || !endDate) {
            showErrorMsg("请选择入住和离店日期");
            return;
        }
        if (startDate >= endDate) {
            showErrorMsg("离店日期须晚于入住日期");
            return;
        }
        var days = calcNights(startDate, endDate);
        if (days < 1) {
            showErrorMsg("日期有误，请重新选择");
            return;
        }
        if (minDays > 0 && days < minDays) {
            showErrorMsg("最少入住" + minDays + "晚");
            return;
        }
        if (maxDays > 0 && days > maxDays) {
            showErrorMsg("最多入住" + maxDays + "晚");
            return;
        }

        setSubmitting(true);
        $.ajax({
            url: "/api/v1.0/orders",
            type: "POST",
            data: JSON.stringify({
                "house_id": houseId,
                "start_date": startDate,
                "end_date": endDate
            }),
            contentType: "application/json",
            dataType: "json",
            success: function (resp) {
                if ("4101" == resp.errno) {
                    location.href = "/home/login.html";
                    return;
                }
                if ("4004" == resp.errno) {
                    showErrorMsg(resp.errmsg || "房间已被抢定，请重新选择日期");
                    setSubmitting(false);
                    return;
                }
                if ("0" == resp.errno) {
                    location.href = "/home/orders.html";
                    return;
                }
                showErrorMsg(resp.errmsg || "下单失败，请稍后重试");
                setSubmitting(false);
            },
            error: function () {
                showErrorMsg("网络异常，请稍后重试");
                setSubmitting(false);
            }
        });
    });
});
