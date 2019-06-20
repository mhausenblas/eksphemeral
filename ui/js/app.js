'use strict';

// the control plane proxy URL:
var cpURL = 'http://localhost:8080';

// how fast to refresh cluster list (5 * 60 * 1000 = every 5 min)
var refreshClusterList= 5*60*1000;

// how fast to refresh cluster details (10 * 1000 = every 10 sec)
var refreshClusterDetails = 10*1000;

$(document).ready(function($){
  clusters();

  // list clusters periodically:
  setInterval(clusters, refreshClusterList);

  // incrementally update cluster headers:
  setInterval(updateClusters, refreshClusterDetails);

  // manually list clusters when user clicks the refresh button:
  $('#clusters > h2').click(function (event) {
    clusters();
  });

  // show cluster details when user clicks 'Details'
  // note: since it's an dynamically added element, needs the .on() form:
  $('body').on('click', 'span.detailsbtn', function () {
    var cID = $(this).parent().attr('id');
    clusterdetail(cID);
  });

  // when user clicks the create button in the right upper corner:
  $('#create').click(function (event) {
    $('#createdialog').show();
  });
  // when user clicks the Go! button in the dialog command row:
  $('#submitcc').click(function (event) {
    $('#createdialog').hide();
    createCluster();
  });
  // when user clicks the Cancel button in the dialog command row:
  $('#cancelcc').click(function (event) {
    $('#createdialog').hide();
  });

  // prolong cluster lifetime for 30min when user clicks 'Prolong'
  // note: since it's an dynamically added element, needs the .on() form:
  $('body').on('click', 'span.prolongbtn', function () {
    var cID = $(this).parent().attr('id');
    var prolongTime = 30;
    prolongCluster(cID, prolongTime);
  });

  $('body').on('click', 'span.showconfbtn', function () {
    var cID = $(this).parent().attr('id');
    var prolongTime = 30;
    prolongCluster(cID, prolongTime);
  });
});

function createCluster() {
    console.info('Calling out to local proxy for cluster creation');
    $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');

    var cname = $('#icname').val();
    var cworkernum = $('#icworkernum').val();
    var cversion = $('#ickversion option:selected').text();
    var ctimeout = $('#ictimeout').val();
    var cowner = $('#icowner').val();
    var clusterspec = { 
      'name': cname, 
      'numworkers': parseInt(cworkernum, 10),
      'kubeversion': cversion, 
      'timeout': parseInt(ctimeout, 10),
      'owner': cowner
    };
    $.ajax({
      type: 'POST',
      url: cpURL+'/create',
      dataType: 'json',
      data: JSON.stringify(clusterspec),
      async: true,
      error: function (d) {
        console.info(d);
        $('#status').html('<div>'+ d.responseText + '</div>');
      },
      success: function (d) {
        if (d != null) {
          console.info(d.responseText);
          $('#status').html('<div>Provisioning cluster with ID '+ d.responseText + ' now! This can take up to 15 minutes, will try to notify you via mail.</div>');
        }
      }
    });
}

function prolongCluster(cID, prolongTime) {
    console.info('Calling out to local proxy for prolonging cluster lifetime');
    $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');

    var clusterprolong = { 
      'id': cID, 
      'ptime': prolongTime,
    };
    $.ajax({
      type: 'POST',
      url: cpURL+'/prolong',
      dataType: 'json',
      data: JSON.stringify(clusterprolong),
      async: true,
      error: function (d) {
        console.info(d.responseText);
        $('#status').html('<div>'+ d.responseText + '</div>');
      },
      success: function (d) {
        if (d != null) {
          console.info(d.responseText);
          $('#status').html('<div>'+ d.responseText + '</div>');
        }
      }
    });
}

function updateClusters(){
  console.info('Scanning cluster list');

  $('div.cluster span.cdlabel').each(function (index, value) {
    var cID = $(this).parent().attr('id');
    var lval = $('#' + cID + ' .cdlabel a').text();
    var ep = '/status?cluster=' + cID;
    console.info('Checking cluster with ID ' + cID + ' with the label ' + lval);
    if (lval == cID){
      $.ajax({
        type: 'GET',
        url: cpURL + ep,
        dataType: 'json',
        async: true,
        error: function (d) {
          console.info(d.responseText);
          $('#status').html('<div>'+ d.responseText + '</div>');
        },
        success: function (d) {
          if (d != null) {
            console.info(d);
            var consoleLink = 'https://console.aws.amazon.com/eks/home?#/clusters/';
            var buffer = '';
            buffer += d.name;
            $('#' + cID + ' .cdlabel a').html(buffer);
            $('#' + cID + ' .cdlabel a').attr('href', consoleLink + d.name);
          }
        }
      })
    }
  });
}

function clusters(){
  var ep = '/status?cluster=*';

  $.ajax({
    type: 'GET',
    url: cpURL + ep,
    dataType: 'json',
    async: true,
    error: function (d) {
      console.info(d.responseText);
      $('#status').html('<div>'+ d.responseText + '</div>');
    },
    success: function (d) {
      if (d != null) {
        console.info(d);
        var buffer = '';
        var consoleURL = "https://console.aws.amazon.com/eks/";
        for (let i = 0; i < d.length; i++) {
          var cID = d[i];
          buffer += '<div class="cluster" id="' + cID + '">';
          buffer += ' <span class="cdlabel"><a href="' + consoleURL + '" target="_blank" rel="noopener">' + cID + '</a></span>';
          buffer += ' <span class="showconfbtn">Show Config</span> <span class="prolongbtn">Prolong</span> <span class="detailsbtn">Details…</span>';
          buffer += '<div class="cdetails"></div>';
          buffer += '</div>';
        }
        $('#clusterdetails').html(buffer);
        $('#status').html('');
      }
    }
  })
}

function clusterdetail(cID) {
  var ep = '/status?cluster='+cID;
  var currentdetails = $('#' + cID + ' .cdetails').text();

  if (currentdetails != '') {
    $('#' + cID + ' .cdetails').toggle();
    return
  }
  $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');
  $.ajax({
    type: 'GET',
    url: cpURL + ep,
    dataType: 'json',
    async: true,
    error: function (d) {
      console.info(d);
      $('#status').html('<div>looking up details for cluster ' + cID + ' failed</div>');
    },
    success: function (d) {
      if (d != null) {
        console.info(d);
        var buffer = '';
        buffer += '<div class="cdfield"><span class="cdtitle">Name:</span> ' + d.name + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Kubernetes version:</span> ' + d.kubeversion + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Number of worker nodes:</span> ' + d.numworkers + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Created at:</span> ' + convertTimestamp(d.created) + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Timeout:</span> ' + d.timeout + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">TTL:</span> ' + d.ttl + ' min left</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Owner:</span> <a href="mailto:' + d.owner + '">' + d.owner + '</a> notified on creation and 5 min before destruction</div>';
        var dbuffer = '';
        dbuffer += '<div class="moarfield"><span class="cdtitle">Status:</span> ' + d.details['status'] + '</div>';
        dbuffer += '<div class="moarfield"><span class="cdtitle">Endpoint:</span> <code>' + d.details['endpoint'] + '</code></div>';
        dbuffer += '<div class="moarfield"><span class="cdtitle">Platform version:</span> ' + d.details['platformv'] + '</div>';
        dbuffer += '<div class="moarfield"><span class="cdtitle">VPC config:</span> ' + d.details['vpcconf'] + '</div>';
        dbuffer += '<div class="moarfield"><span class="cdtitle">IAM role:</span> <code>' + d.details['iamrole'] + '</code></div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Cluster summary:</span> ' + dbuffer + '</div>';
        $('#' + cID + ' .cdetails').html(buffer);
        $('#status').html('');
      }
    }
  })
}

// as per https://gist.github.com/kmaida/6045266
function convertTimestamp(timestamp) {
  var d = new Date(timestamp * 1000),	// Convert the passed timestamp to milliseconds
		yyyy = d.getFullYear(),
		mm = ('0' + (d.getMonth() + 1)).slice(-2),	// Months are zero based. Add leading 0.
		dd = ('0' + d.getDate()).slice(-2),			// Add leading 0.
		hh = d.getHours(),
		h = hh,
		min = ('0' + d.getMinutes()).slice(-2),		// Add leading 0.
		ampm = 'AM',
		time;
			
	if (hh > 12) {
		h = hh - 12;
		ampm = 'PM';
	} else if (hh === 12) {
		h = 12;
		ampm = 'PM';
	} else if (hh == 0) {
		h = 12;
	}
	
	// ie: 2013-02-18, 8:35 AM	
	time = yyyy + '-' + mm + '-' + dd + ', ' + h + ':' + min + ' ' + ampm;
		
	return time;
}
