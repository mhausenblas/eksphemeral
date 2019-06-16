'use strict';

var cpURL = 'https://nswn7lkjbk.execute-api.us-east-2.amazonaws.com/Prod';


$(document).ready(function($){
  $('#create').click(function(event) {
    var ep = '/status/*';
    $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');
    $.ajax({
        type: "GET",
        url: cpURL + ep,
        dataType: 'json',
        async: false,
        error: function (d) {
          console.info(d);
          $('#status').html('<div>control plane seems down</div>');
        },
        success: function (d) {
          console.info(d);
          // $('#clusterdetails').html(d);
          $('#status').html('');
        }
    })
  });
});