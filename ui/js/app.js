'use strict';

$(document).ready(function($){
  $('#create').click(function(event) {
    var ep = 'abc';
    $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');
    $.ajax({
        type: "GET",
        url: '/v1/explorer?endpoint='+encodeURIComponent(ep),
        dataType: 'json',
        async: false,
        data: '{"endpoint": "' + ep +'"}',
        error: function (d) {
          console.info(d);
          $('#clusterdetails').html('');
          $('#status').html('<div>control plane seems down</div>');
        },
        success: function (d) {
          console.info(d);
        }
    })
  });
});