$(document).ready( function(){
	var post_height = $(".main-body-post").height();
	var post_detail_height = $(".main-body-post-detail-post").height();
	$(".main-body-post-detail-post").css({ top : ((post_height-(post_detail_height+40))/2)+"px" });
});
