function hrefBack() {
    history.go(-1);
}

function decodeQuery(){
    var search = decodeURI(document.location.search);
    return search.replace(/(^\?)/, '').split('&').reduce(function(result, item){
        values = item.split('=');
        result[values[0]] = values[1];
        return result;
    }, {});
}

var FACILITY_MAP = {
    "1": "无线网络",
    "2": "热水淋浴",
    "3": "空调",
    "4": "暖气",
    "5": "允许吸烟",
    "6": "饮水设备",
    "7": "牙具",
    "8": "香皂",
    "9": "拖鞋",
    "10": "手纸",
    "11": "毛巾",
    "12": "沐浴露、洗发露",
    "13": "冰箱",
    "14": "洗衣机",
    "15": "电梯",
    "16": "允许做饭",
    "17": "允许带宠物",
    "18": "允许聚会",
    "19": "门禁系统",
    "20": "停车位",
    "21": "有线网络",
    "22": "电视",
    "23": "浴缸"
};

function mapFacilities(ids) {
    var list = [];
    if (!ids || !ids.length) {
        return list;
    }
    for (var i = 0; i < ids.length; i++) {
        var id = String(ids[i]);
        list.push({
            id: id,
            name: FACILITY_MAP[id] || ("设施" + id)
        });
    }
    return list;
}

$(document).ready(function(){
    var queryData = decodeQuery();
    var houseId = queryData["id"];
    if (!houseId) {
        $(".detail-error").text("缺少房屋编号").show();
        return;
    }

    $.get("/api/v1.0/houses/" + houseId, function(resp){
        if ("0" == resp.errno && resp.data && resp.data.house) {
            $(".detail-error").hide();
            var house = resp.data.house;
            var imgUrls = house.img_urls || [];
            if (!imgUrls.length) {
                imgUrls = ["/home/images/home01.jpg"];
            }

            $(".swiper-container").html(template("house-image-tmpl", {
                img_urls: imgUrls
            }));
            $(".detail-con").html(template("house-detail-tmpl", {
                house: house,
                facilities: mapFacilities(house.facilities)
            }));

            $(".book-price-num").text(Number(house.price || 0).toFixed(0));

            // 非房东（含游客 user_id=0）显示预订底栏
            if (resp.data.user_id != house.user_id) {
                var $btn = $(".book-house");
                if (!resp.data.user_id) {
                    $btn.attr("href", "/home/login.html");
                    $btn.text("去登录");
                    $btn.removeClass("book-house-primary").addClass("book-house-ghost");
                } else {
                    $btn.attr("href", "/home/booking.html?hid=" + house.hid);
                    $btn.text("即刻预定");
                    $btn.removeClass("book-house-ghost").addClass("book-house-primary");
                }
                $(".book-bar").css("display", "flex");
            }

            var slideCount = imgUrls.length;
            new Swiper ('.swiper-container', {
                loop: slideCount > 1,
                autoplay: slideCount > 1 ? 2000 : false,
                autoplayDisableOnInteraction: false,
                pagination: '.swiper-pagination',
                paginationType: 'fraction'
            });
        } else {
            $(".detail-error").text((resp && resp.errmsg) ? resp.errmsg : "房屋不存在或加载失败").show();
            $(".swiper-container").hide();
            $(".detail-con").hide();
        }
    }).fail(function(){
        $(".detail-error").text("网络异常，请稍后重试").show();
        $(".swiper-container").hide();
        $(".detail-con").hide();
    });
})
