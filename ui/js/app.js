'use strict';

var cpURL = 'EKSPHEMERAL_URL';


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
          var buffer = "";
          var consoleURL = "https://console.aws.amazon.com/eks/home";
          for (let i = 0; i < d.length; i++) {
            var cID = d[i];
            buffer += '<a href="' + consoleURL + ' target="_blank" rel="noopener">' + cID + '</a>';
            buffer += '<span class="ttl">42 min</span> <span class="details">Details…</span>';
          }
          $('#clusterdetails').html(buffer);
          $('#status').html('');
        }
    })
  });
});