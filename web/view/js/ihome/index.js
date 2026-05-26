//模态框居中的控制
function centerModals(){
    $('.modal').each(function(i){   //遍历每一个模态框
        var $clone = $(this).clone().css('display', 'block').appendTo('body');    
        var top = Math.round(($clone.height() - $clone.find('.modal-content').height()) / 2);
        top = top > 0 ? top : 0;
        $clone.remove();
        $(this).find('.modal-content').css("margin-top", top-30);  //修正原先已经有的30个像素
    });
}

function setStartDate() {
    var startDate = $("#start-date-input").val();
    if (startDate) {
        $(".search-btn").attr("start-date", startDate);
        $("#start-date-btn").html(startDate);
        $("#end-date").datepicker("destroy");
        $("#end-date-btn").html("离开日期");
        $("#end-date-input").val("");
        $(".search-btn").attr("end-date", "");
        $("#end-date").datepicker({
            language: "zh-CN",
            keyboardNavigation: false,
            startDate: startDate,
            format: "yyyy-mm-dd"
        });
        $("#end-date").on("changeDate", function() {
            $("#end-date-input").val(
                $(this).datepicker("getFormattedDate")
            );
        });
        $(".end-date").show();
    }
    $("#start-date-modal").modal("hide");
}

function setEndDate() {
    var endDate = $("#end-date-input").val();
    if (endDate) {
        $(".search-btn").attr("end-date", endDate);
        $("#end-date-btn").html(endDate);
    }
    $("#end-date-modal").modal("hide");
}




function goToSearchPage(th) {
    var url = "/home/search.html?";
    url += ("aid=" + $(th).attr("area-id"));
    url += "&";
    var areaName = $(th).attr("area-name");
    if (undefined == areaName) areaName="";
    url += ("aname=" + areaName);
    url += "&";
    url += ("sd=" + $(th).attr("start-date"));
    url += "&";
    url += ("ed=" + $(th).attr("end-date"));
    location.href = url;
}

$(document).ready(function(){
    // 检查用户的登录状态
    $.get("/api/v1.0/session", function(resp) {
        if ("0" == resp.errno) {
            $(".top-bar>.user-info>.user-name").html(resp.data.name);
            $(".top-bar>.user-info").show();
            //yincang

        } else {
            $(".top-bar>.register-login").show();
        }
    }, "json");

    function normalizeBannerImgUrl(imgUrl, index) {
        var localImages = [
            "/home/images/home01.jpg",
            "/home/images/home02.jpg",
            "/home/images/home03.jpg"
        ];
        if (imgUrl) {
            if (imgUrl.indexOf("/home/images/") === 0) {
                return imgUrl;
            }
            if (imgUrl.indexOf("./images/") === 0) {
                return "/home/images/" + imgUrl.replace("./images/", "");
            }
            var match = imgUrl.match(/home0[1-3]\.jpg/);
            if (match) {
                return "/home/images/" + match[0];
            }
        }
        return localImages[index % localImages.length];
    }

    function initIndexSwiper(houses) {
        if (!houses || houses.length === 0) {
            console.warn("首页轮播无房屋数据");
            return;
        }
        for (var i = 0; i < houses.length; i++) {
            houses[i].img_url = normalizeBannerImgUrl(houses[i].img_url, i);
        }
        $(".swiper-wrapper").html(
            template("swiper-houses-tmpl", { houses: houses })
        );
        var slideCount = houses.length;
        var mySwiper = new Swiper(".swiper-container", {
            loop: slideCount > 1,
            autoplay: slideCount > 1 ? 2000 : false,
            autoplayDisableOnInteraction: false,
            pagination: ".swiper-pagination",
            paginationClickable: true,
            observer: true,
            observeParents: true
        });
        mySwiper.update();
    }

    // 从接口读取数据库房屋信息，图片使用本地 /home/images/home0x.jpg
    $.get("/api/v1.0/house/index", function(resp) {
        if ("0" == resp.errno && resp.data && resp.data.houses && resp.data.houses.length > 0) {
            initIndexSwiper(resp.data.houses);
        } else {
            console.warn("获取首页轮播失败", resp);
        }
    }, "json");

    // 获取城区信息
    $.get("/api/v1.0/areas", function(resp){
        if ("0" == resp.errno) {
            $(".area-list").html(template("area-list-tmpl", {areas:resp.data}));

            $(".area-list a").click(function(e){
                $("#area-btn").html($(this).html());
                $(".search-btn").attr("area-id", $(this).attr("area-id"));
                $(".search-btn").attr("area-name", $(this).html());
                $("#area-modal").modal("hide");
            });
        }
    });
    $('.modal').on('show.bs.modal', centerModals);      //当模态框出现的时候
    $(window).on('resize', centerModals);               //当窗口大小变化的时候
    $("#start-date").datepicker({
        language: "zh-CN",
        keyboardNavigation: false,
        startDate: "today",
        format: "yyyy-mm-dd"
    });
    $("#start-date").on("changeDate", function() {
        var date = $(this).datepicker("getFormattedDate");
        $("#start-date-input").val(date);
    });
})