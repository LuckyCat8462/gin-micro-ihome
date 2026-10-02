var cur_page = 1; // 当前页
var next_page = 1; // 下一页
var total_page = 1;  // 总页数
var house_data_querying = false;   // 是否正在向后台获取数据

// 解析url中的查询字符串
function decodeQuery(){
    var search = decodeURI(document.location.search);
    return search.replace(/(^\?)/, '').split('&').reduce(function(result, item){
        var values = item.split('=');
        result[values[0]] = values[1];
        return result;
    }, {});
}

// 更新用户点选的筛选条件
function updateFilterDateDisplay() {
    var startDate = $("#start-date").val();
    var endDate = $("#end-date").val();
    var $filterDateTitle = $(".filter-title-bar>.filter-title").eq(0).children("span").eq(0);
    if (startDate && endDate) {
        var text = startDate.substr(5) + "/" + endDate.substr(5);
        $filterDateTitle.html(text);
    } else {
        $filterDateTitle.html("入住日期");
    }
}

function renewHouseList() {
    cur_page = 1;
    next_page = 1;
    total_page = 1;
    updateHouseData("renew");
}

// 更新房源列表信息
// action表示从后端请求的数据在前端的展示方式
// 默认采用追加方式
// action=renew 代表页面数据清空从新展示
function updateHouseData(action) {
    var areaId = $(".filter-area>li.active").attr("area-id");
    if (undefined == areaId) areaId = "";
    var startDate = $("#start-date").val() || "";
    var endDate = $("#end-date").val() || "";
    var sortKey = $(".filter-sort>li.active").attr("sort-key") || "new";

    if ((startDate && !endDate) || (!startDate && endDate)) {
        house_data_querying = false;
        alert("请同时选择入住和离店日期");
        return;
    }
    if (startDate && endDate && startDate >= endDate) {
        house_data_querying = false;
        alert("离店日期须晚于入住日期");
        return;
    }

    house_data_querying = true;
    var params = {
        aid: areaId,
        sd: startDate,
        ed: endDate,
        sk: sortKey,
        p: next_page
    };
    $.get("/api/v1.0/houses", params, function(resp){
        if ("0" == resp.errno) {
            var pageTotal = (resp.data && resp.data.total_page != null) ? String(resp.data.total_page) : "0";
            if ("0" == pageTotal) {
                total_page = 0;
                $(".house-list").html("暂时没有符合您查询的房屋信息。");
            } else {
                total_page = parseInt(pageTotal, 10) || 1;
                var houses = (resp.data && resp.data.houses) ? resp.data.houses : [];
                if ("renew" == action) {
                    cur_page = 1;
                    $(".house-list").html(template("house-list-tmpl", {houses: houses}));
                } else {
                    cur_page = next_page;
                    $(".house-list").append(template("house-list-tmpl", {houses: houses}));
                }
            }
        } else {
            if ("renew" == action) {
                $(".house-list").html((resp && resp.errmsg) ? resp.errmsg : "加载房源失败");
            } else {
                alert((resp && resp.errmsg) ? resp.errmsg : "加载房源失败");
            }
        }
    }).fail(function(){
        if ("renew" == action) {
            $(".house-list").html("网络异常，请稍后重试");
        } else {
            alert("网络异常，请稍后重试");
        }
    }).always(function(){
        house_data_querying = false;
    });
}

$(document).ready(function(){
    var queryData = decodeQuery();
    var startDate = queryData["sd"] || "";
    var endDate = queryData["ed"] || "";
    $("#start-date").val(startDate);
    $("#end-date").val(endDate);
    updateFilterDateDisplay();
    var areaName = queryData["aname"];
    if (!areaName) areaName = "位置区域";
    $(".filter-title-bar>.filter-title").eq(1).children("span").eq(0).html(areaName);

    var sortKey = queryData["sk"] || "new";
    $(".filter-sort>li").each(function(){
        if ($(this).attr("sort-key") == sortKey) {
            $(this).addClass("active").siblings("li").removeClass("active");
            $(".filter-title-bar>.filter-title").eq(2).children("span").eq(0).html($(this).html());
        }
    });

    function bindScrollLoad() {
        var windowHeight = $(window).height();
        window.onscroll = function(){
            var b = document.documentElement.scrollTop==0? document.body.scrollTop : document.documentElement.scrollTop;
            var c = document.documentElement.scrollTop==0? document.body.scrollHeight : document.documentElement.scrollHeight;
            if(c-b<windowHeight+50){
                if (!house_data_querying && cur_page < total_page) {
                    next_page = cur_page + 1;
                    updateHouseData();
                }
            }
        };
    }

    // 区域与列表解耦：先拉区域，失败仍可搜房
    $.get("/api/v1.0/areas", function(data){
        if ("0" == data.errno) {
            var areaId = queryData["aid"];
            if (areaId) {
                for (var i=0; i<data.data.length; i++) {
                    if (String(data.data[i].aid) == String(areaId)) {
                        $(".filter-area").append('<li area-id="'+ data.data[i].aid+'" class="active">'+ data.data[i].aname+'</li>');
                    } else {
                        $(".filter-area").append('<li area-id="'+ data.data[i].aid+'">'+ data.data[i].aname+'</li>');
                    }
                }
            } else {
                for (var i=0; i<data.data.length; i++) {
                    $(".filter-area").append('<li area-id="'+ data.data[i].aid+'">'+ data.data[i].aname+'</li>');
                }
            }
        }
    }).always(function(){
        renewHouseList();
        bindScrollLoad();
    });

    $(".input-daterange").datepicker({
        format: "yyyy-mm-dd",
        startDate: "today",
        language: "zh-CN",
        autoclose: true
    });
    var $filterItem = $(".filter-item-bar>.filter-item");
    $(".filter-title-bar").on("click", ".filter-title", function(e){
        var index = $(this).index();
        if (!$filterItem.eq(index).hasClass("active")) {
            $(this).children("span").children("i").removeClass("fa-angle-down").addClass("fa-angle-up");
            $(this).siblings(".filter-title").children("span").children("i").removeClass("fa-angle-up").addClass("fa-angle-down");
            $filterItem.eq(index).addClass("active").siblings(".filter-item").removeClass("active");
            $(".display-mask").show();
        } else {
            $(this).children("span").children("i").removeClass("fa-angle-up").addClass("fa-angle-down");
            $filterItem.eq(index).removeClass('active');
            $(".display-mask").hide();
            updateFilterDateDisplay();
            renewHouseList();
        }
    });
    $(".display-mask").on("click", function(e) {
        $(this).hide();
        $filterItem.removeClass('active');
        updateFilterDateDisplay();
        renewHouseList();
    });
    $(".filter-item-bar>.filter-area").on("click", "li", function(e) {
        if (!$(this).hasClass("active")) {
            $(this).addClass("active");
            $(this).siblings("li").removeClass("active");
            $(".filter-title-bar>.filter-title").eq(1).children("span").eq(0).html($(this).html());
        } else {
            $(this).removeClass("active");
            $(".filter-title-bar>.filter-title").eq(1).children("span").eq(0).html("位置区域");
        }
        renewHouseList();
    });
    $(".filter-item-bar>.filter-sort").on("click", "li", function(e) {
        if (!$(this).hasClass("active")) {
            $(this).addClass("active");
            $(this).siblings("li").removeClass("active");
            $(".filter-title-bar>.filter-title").eq(2).children("span").eq(0).html($(this).html());
            renewHouseList();
        }
    });
    $(".input-daterange").on("changeDate", function(){
        updateFilterDateDisplay();
    });
})
